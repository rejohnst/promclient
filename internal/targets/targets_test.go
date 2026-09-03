package targets

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/api"
)

func TestTargetsCountPrintsOnlyCount(t *testing.T) {
	server := newTargetsTestServer(t)
	var out, errOut bytes.Buffer

	err := Targets(context.Background(), server.client, &out, &errOut, &TargetArgs{Count: true, Job: "node"})
	if err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}

	if got, want := out.String(), "2\n"; got != want {
		t.Fatalf("unexpected count output: got %q want %q", got, want)
	}
}

func TestTargetsDownCountAppliesFilters(t *testing.T) {
	server := newTargetsTestServer(t)
	var out, errOut bytes.Buffer

	err := Targets(context.Background(), server.client, &out, &errOut, &TargetArgs{
		Count: true,
		Down:  true,
		Job:   "node",
	})
	if err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}

	if got, want := out.String(), "1\n"; got != want {
		t.Fatalf("unexpected down count output: got %q want %q", got, want)
	}
}

func TestTargetsJobFilterAppliesToActiveTargets(t *testing.T) {
	server := newTargetsTestServer(t)
	var out, errOut bytes.Buffer

	err := Targets(context.Background(), server.client, &out, &errOut, &TargetArgs{Job: "node"})
	if err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "http://node-a:9100/metrics") {
		t.Fatalf("expected matching active target, got:\n%s", got)
	}
	if strings.Contains(got, "api:9000") {
		t.Fatalf("expected job filter to exclude api, got:\n%s", got)
	}
}

// Collecting dropped targets can exceed the 2 GiB gRPC message limit behind
// Thanos Query and fail the whole request, so they are never requested.
func TestTargetsAlwaysRequestsActiveStateOnly(t *testing.T) {
	server := newTargetsTestServer(t)
	var out, errOut bytes.Buffer

	err := Targets(context.Background(), server.client, &out, &errOut, &TargetArgs{})
	if err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}

	if got, want := server.lastQuery().Get("state"), "active"; got != want {
		t.Fatalf("unexpected state parameter: got %q want %q", got, want)
	}
}

func TestTargetsOmitsDroppedSection(t *testing.T) {
	server := newTargetsTestServer(t)
	var out, errOut bytes.Buffer

	err := Targets(context.Background(), server.client, &out, &errOut, &TargetArgs{})
	if err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}

	if got := out.String(); strings.Contains(got, "Dropped targets:") {
		t.Fatalf("expected no dropped-target section, got:\n%s", got)
	}
}

func TestTargetsNeverSendsScrapePool(t *testing.T) {
	server := newTargetsTestServer(t)
	var out, errOut bytes.Buffer

	err := Targets(context.Background(), server.client, &out, &errOut, &TargetArgs{Job: "node"})
	if err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}

	// A scrape pool is named after its scrape config's job_name, which under
	// prometheus-operator never matches the job label.
	if got, ok := server.lastQuery()["scrapePool"]; ok {
		t.Fatalf("expected no scrapePool parameter, got %q", got)
	}
}

// Thanos Query reports a failed store fan-out as an empty result plus
// warnings, which is otherwise indistinguishable from having no targets.
func TestTargetsReportsWarningsOnStderr(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/targets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"warnings": ["receiving targets from targets client: rpc error: code = ResourceExhausted"],
			"data": {"activeTargets": [], "droppedTargets": []}
		}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := api.NewClient(api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create Prometheus client: %v", err)
	}

	var out, errOut bytes.Buffer
	if err := Targets(context.Background(), client, &out, &errOut, &TargetArgs{Count: true}); err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}

	if got, want := out.String(), "0\n"; got != want {
		t.Fatalf("unexpected count output: got %q want %q", got, want)
	}
	if got := errOut.String(); !strings.Contains(got, "ResourceExhausted") {
		t.Fatalf("expected warning on stderr, got %q", got)
	}
}

func TestTargetsReportsAPIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/targets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"status":"error","errorType":"bad_data","error":"invalid parameter"}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := api.NewClient(api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create Prometheus client: %v", err)
	}

	var out, errOut bytes.Buffer
	err = Targets(context.Background(), client, &out, &errOut, &TargetArgs{})
	if err == nil {
		t.Fatal("expected Targets to return an error")
	}
	if got := err.Error(); !strings.Contains(got, "invalid parameter") {
		t.Fatalf("unexpected error: %q", got)
	}
}

// targetsTestServer serves a fixed targets payload and records the query
// parameters of the most recent request.
type targetsTestServer struct {
	client api.Client

	mu    sync.Mutex
	query url.Values
}

func (s *targetsTestServer) lastQuery() url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.query
}

func newTargetsTestServer(t *testing.T) *targetsTestServer {
	t.Helper()

	testServer := &targetsTestServer{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/targets", func(w http.ResponseWriter, r *http.Request) {
		testServer.mu.Lock()
		testServer.query = r.URL.Query()
		testServer.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"activeTargets": [
					{
						"discoveredLabels": {"job": "node", "__address__": "node-a:9100"},
						"labels": {"job": "node", "instance": "node-a:9100"},
						"scrapePool": "node",
						"scrapeUrl": "http://node-a:9100/metrics",
						"globalUrl": "http://node-a:9100/metrics",
						"lastError": "",
						"lastScrape": "2026-04-29T00:00:00Z",
						"lastScrapeDuration": 0.01,
						"health": "up"
					},
					{
						"discoveredLabels": {"job": "node", "__address__": "node-b:9100"},
						"labels": {"job": "node", "instance": "node-b:9100"},
						"scrapePool": "node",
						"scrapeUrl": "http://node-b:9100/metrics",
						"globalUrl": "http://node-b:9100/metrics",
						"lastError": "connection refused",
						"lastScrape": "2026-04-29T00:00:00Z",
						"lastScrapeDuration": 0.01,
						"health": "down"
					},
					{
						"discoveredLabels": {"job": "api", "__address__": "api:9000"},
						"labels": {"job": "api", "instance": "api:9000"},
						"scrapePool": "api",
						"scrapeUrl": "http://api:9000/metrics",
						"globalUrl": "http://api:9000/metrics",
						"lastError": "timeout",
						"lastScrape": "2026-04-29T00:00:00Z",
						"lastScrapeDuration": 0.01,
						"health": "down"
					}
				],
				"droppedTargets": [
					{"discoveredLabels": {"job": "node", "__address__": "node-c:9100"}},
					{"discoveredLabels": {"job": "api", "__address__": "api-old:9000"}}
				]
			}
		}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := api.NewClient(api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create Prometheus client: %v", err)
	}
	testServer.client = client

	return testServer
}

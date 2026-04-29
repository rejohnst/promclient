package targets

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func TestTargetsCountPrintsOnlyCount(t *testing.T) {
	api := newTargetsTestAPI(t)
	var out bytes.Buffer

	err := Targets(context.Background(), api, &out, &TargetArgs{Count: true, Job: "node"})
	if err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}

	if got, want := out.String(), "2\n"; got != want {
		t.Fatalf("unexpected count output: got %q want %q", got, want)
	}
}

func TestTargetsDownCountAppliesFilters(t *testing.T) {
	api := newTargetsTestAPI(t)
	var out bytes.Buffer

	err := Targets(context.Background(), api, &out, &TargetArgs{
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

func TestTargetsJobFilterAppliesToDroppedTargets(t *testing.T) {
	api := newTargetsTestAPI(t)
	var out bytes.Buffer

	err := Targets(context.Background(), api, &out, &TargetArgs{Job: "node"})
	if err != nil {
		t.Fatalf("Targets returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Dropped targets:\nJob: node\n") {
		t.Fatalf("expected matching dropped target, got:\n%s", got)
	}
	if strings.Contains(got, "Job: api\n") {
		t.Fatalf("expected dropped target job filter to exclude api, got:\n%s", got)
	}
}

func newTargetsTestAPI(t *testing.T) v1.API {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/targets", func(w http.ResponseWriter, r *http.Request) {
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
	return v1.NewAPI(client)
}

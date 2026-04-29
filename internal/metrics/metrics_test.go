package metrics

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func TestMetricsCountUsesEscapedJobMatcher(t *testing.T) {
	const job = `node"exporter`
	const wantMatcher = `{job="node\"exporter"}`

	var gotMatcher string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/targets/metadata", func(w http.ResponseWriter, r *http.Request) {
		gotMatcher = r.URL.Query().Get("match_target")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": [
				{
					"target": {"job": "node\"exporter"},
					"metric": "up",
					"type": "gauge",
					"help": "Target health, with comma",
					"unit": ""
				}
			]
		}`))
	})

	api := newMetricsTestAPI(t, mux)
	var out bytes.Buffer
	err := Metrics(context.Background(), api, &out, &MetricArgs{Count: true, Job: job})
	if err != nil {
		t.Fatalf("Metrics returned error: %v", err)
	}

	if gotMatcher != wantMatcher {
		t.Fatalf("unexpected match_target: got %q want %q", gotMatcher, wantMatcher)
	}
	if got, want := out.String(), "1\n"; got != want {
		t.Fatalf("unexpected count output: got %q want %q", got, want)
	}
}

func TestMetricsCSVPreservesCommasAndQuotes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/targets/metadata", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": [
				{
					"target": {"job": "node"},
					"metric": "node_metric",
					"type": "counter",
					"help": "help with comma, and \"quote\"",
					"unit": ""
				}
			]
		}`))
	})

	api := newMetricsTestAPI(t, mux)
	var out bytes.Buffer
	err := Metrics(context.Background(), api, &out, &MetricArgs{CSV: true, Job: "node"})
	if err != nil {
		t.Fatalf("Metrics returned error: %v", err)
	}

	got := out.String()
	want := "node,node_metric,\"help with comma, and \"\"quote\"\"\",counter\n"
	if got != want {
		t.Fatalf("unexpected CSV output: got %q want %q", got, want)
	}
}

func newMetricsTestAPI(t *testing.T, handler http.Handler) v1.API {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := api.NewClient(api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create Prometheus client: %v", err)
	}
	return v1.NewAPI(client)
}

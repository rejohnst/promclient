package alerts

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

func TestAlertsCountAppliesDisplayFilters(t *testing.T) {
	api := newAlertsTestAPI(t)
	var out bytes.Buffer

	err := Alerts(context.Background(), api, &out, &AlertArgs{
		Count:    true,
		Severity: "critical",
	})
	if err != nil {
		t.Fatalf("Alerts returned error: %v", err)
	}

	if got, want := out.String(), "1\n"; got != want {
		t.Fatalf("unexpected count output: got %q want %q", got, want)
	}
}

func TestAlertsOutputAppliesFilters(t *testing.T) {
	api := newAlertsTestAPI(t)
	var out bytes.Buffer

	err := Alerts(context.Background(), api, &out, &AlertArgs{Severity: "critical"})
	if err != nil {
		t.Fatalf("Alerts returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"alert: HighCPU", "severity: critical", "critical cpu"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"Watchdog", "pending cpu", "DiskFull"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("expected output to exclude %q, got:\n%s", unwanted, got)
		}
	}
}

func newAlertsTestAPI(t *testing.T) v1.API {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"alerts": [
					{
						"labels": {"alertname": "Watchdog", "severity": "none"},
						"annotations": {"message": "watchdog"},
						"state": "firing",
						"activeAt": "2026-04-29T00:00:00Z",
						"value": "1e+00"
					},
					{
						"labels": {"alertname": "HighCPU", "severity": "critical"},
						"annotations": {"message": "pending cpu"},
						"state": "pending",
						"activeAt": "2026-04-29T00:00:00Z",
						"value": "1e+00"
					},
					{
						"labels": {"alertname": "HighCPU", "severity": "critical"},
						"annotations": {"message": "critical cpu"},
						"state": "firing",
						"activeAt": "2026-04-29T00:00:00Z",
						"value": "1e+00"
					},
					{
						"labels": {"alertname": "DiskFull", "severity": "warning"},
						"annotations": {"message": "disk warning"},
						"state": "firing",
						"activeAt": "2026-04-29T00:00:00Z",
						"value": "1e+00"
					}
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

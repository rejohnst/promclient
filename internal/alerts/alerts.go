package alerts

import (
	"context"
	"fmt"
	"io"
	"sort"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type AlertArgs struct {
	// Severity filters results by severity when set.
	Severity string
	// Count prints only a count of matching alerts.
	Count bool
}

type alertGroup struct {
	severity     string
	descriptions []string
}

// Alerts prints active Prometheus alerts.
func Alerts(ctx context.Context, api v1.API, w io.Writer, args *AlertArgs) error {
	result, err := api.Alerts(ctx)
	if err != nil {
		return fmt.Errorf("retrieve alerts: %w", err)
	}

	grouped := make(map[string]*alertGroup)
	count := 0
	for _, alert := range result.Alerts {
		if !matchesAlert(alert, args.Severity) {
			continue
		}

		count++
		key := string(alert.Labels["alertname"])
		entry, ok := grouped[key]
		if !ok {
			entry = &alertGroup{severity: string(alert.Labels["severity"])}
			grouped[key] = entry
		}
		entry.descriptions = append(entry.descriptions, string(alert.Annotations["message"]))
	}

	if args.Count {
		_, err := fmt.Fprintf(w, "%d\n", count)
		return err
	}

	if len(grouped) == 0 {
		return nil
	}

	fmt.Fprintln(w)
	names := make([]string, 0, len(grouped))
	for name := range grouped {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		alert := grouped[name]
		fmt.Fprintf(w, "alert: %s\n", name)
		fmt.Fprintf(w, "severity: %s\n", alert.severity)
		for _, description := range alert.descriptions {
			fmt.Fprintf(w, "  %s\n", description)
		}
		fmt.Fprintln(w)
	}

	return nil
}

func matchesAlert(alert v1.Alert, severity string) bool {
	if alert.Labels["alertname"] == "Watchdog" || alert.State != v1.AlertStateFiring {
		return false
	}
	if severity != "" && alert.Labels["severity"] != model.LabelValue(severity) {
		return false
	}
	return true
}

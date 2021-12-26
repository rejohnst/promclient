package alerts

import (
	"context"
	"fmt"
	"os"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type AlertArgs struct {
	// if true, only output information on critical alerts
	Critical bool
	// if true, just print a count of the matching alerts found
	Count bool
}

type Alert struct {
	paSeverity string
	paDescs    []string
}

//
// Dump all active alerts
//
func Alerts(ctx context.Context, api v1.API, args *AlertArgs) {
	result, err := api.Alerts(ctx)
	alerts := make(map[string]*Alert)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of alerts: %v\n", err)
		os.Exit(1)
	}

	if args.Count {
		fmt.Printf("%d\n", len(result.Alerts))
		return
	}

	fmt.Printf("\n")
	for _, alert := range result.Alerts {
		if alert.Labels["alertname"] == "Watchdog" {
			continue
		}
		if args.Critical && alert.Labels["severity"] != "critical" {
			continue
		}
		key := string(alert.Labels["alertname"])
		val, ok := alerts[key]
		if !ok {
			var newAlert Alert
			newAlert.paSeverity = string(alert.Labels["severity"])
			newAlert.paDescs = append(newAlert.paDescs, string(alert.Annotations["message"]))
			alerts[key] = &newAlert
		} else {
			val.paDescs = append(alerts[key].paDescs, string(alert.Annotations["message"]))
		}
	}
	for k, v := range alerts {
		fmt.Printf("alert: %s\n", k)
		fmt.Printf("severity: %s\n", v.paSeverity)
		for i := range v.paDescs {
			fmt.Printf("  %s\n", v.paDescs[i])
		}
		fmt.Printf("\n")
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type promClient struct {
	pcAPI     v1.API
	pcCtx     context.Context
	pcVerbose bool
}

func promTargets(client *promClient, job *string, active bool, down bool) {
	result, err := client.pcAPI.Targets(client.pcCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of targets: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Active targets\n")
	fmt.Printf("==============\n")
	for _, target := range result.Active {

		if *job != "" && *job != string(target.Labels["job"]) {
			continue
		}
		if down && target.Health != v1.HealthBad {
			continue
		}

		fmt.Printf("%-20s %s\n", "Scrape URL:", target.ScrapeURL)
		fmt.Printf("%-20s %s\n", "Jobs:", target.Labels["job"])
		if target.Labels["pod"] != "" {
			fmt.Printf("%-20s %s\n", "Pod:", target.Labels["pod"])
		}
		fmt.Printf("%-20s %s\n", "State:", target.Health)
		if target.Health == v1.HealthBad {
			fmt.Printf("%-20s %s\n", "Error:", target.LastError)
		}

		if client.pcVerbose {
			for k, v := range target.DiscoveredLabels {
				fmt.Printf("%-20s %s\n", k, v)
			}
		}
		fmt.Printf("\n")
	}

	if active {
		return
	}

	fmt.Printf("Dropped targets:\n")
	for _, target := range result.Dropped {
		if *job != "" && *job != target.DiscoveredLabels["job"] {
			continue
		}
		fmt.Printf("Job: %s\n", target.DiscoveredLabels["job"])
		if client.pcVerbose {
			for k, v := range target.DiscoveredLabels {
				fmt.Printf("%s: %s\n", k, v)
			}
		}
		fmt.Printf("\n")
	}
}

func promAlerts(client *promClient) {
	result, err := client.pcAPI.Alerts(client.pcCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of alerts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n")
	for _, alert := range result.Alerts {
		fmt.Printf("alert: %s\n", alert.Labels["alertname"])
		fmt.Printf("message: %s\n", alert.Annotations["message"])
		fmt.Printf("severity: %s\n", alert.Labels["severity"])
		fmt.Printf("\n")
	}
}

func main() {
	var client promClient
	var promURL, cmd, job *string
	var verbose, active, down *bool
	var cancel context.CancelFunc

	promURL = flag.String("promurl", "", "URL of Prometheus server")
	cmd = flag.String("command", "", "<targets|alerts>")
	job = flag.String("job", "", "show only targets from specified job")
	active = flag.Bool("active", false, "only display active targets")
	down = flag.Bool("down", false, "only display active targets that are down (implies -active)")
	verbose = flag.Bool("verbose", false, "enable verbose mode")
	flag.Parse()

	if *promURL == "" {
		fmt.Fprintf(os.Stderr, "-promurl is a required argument\n\n")
		flag.Usage()
		os.Exit(2)
	}
	if *cmd == "" {
		fmt.Fprintf(os.Stderr, "-command is a required argument\n\n")
		flag.Usage()
		os.Exit(2)
	}
	if *down {
		*active = true
	}
	apiclient, err := api.NewClient(api.Config{
		Address: *promURL,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	client.pcVerbose = *verbose
	client.pcAPI = v1.NewAPI(apiclient)
	client.pcCtx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

	switch *cmd {
	case "alerts":
		promAlerts(&client)
	case "targets":
		promTargets(&client, job, *active, *down)
	default:
		fmt.Fprintf(os.Stderr, "Invalid command: %s\n", *cmd)
		os.Exit(2)
	}

	cancel()
	os.Exit(0)
}

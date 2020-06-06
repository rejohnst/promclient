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
	pcAPI		v1.API
	pcCtx		context.Context
	pcVerbose	bool
}

func promTargets(client *promClient, job *string) {
	result, err := client.pcAPI.Targets(client.pcCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of targets: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Active targets:\n");
	for _, target := range result.Active {
		if *job != "" && *job != string(target.Labels["job"]) {
			continue;
		}
		fmt.Printf("Scrape URL: %s\n", target.ScrapeURL)
		fmt.Printf("Job: %s\n", target.Labels["job"])
		if target.Labels["pod"] != "" {
			fmt.Printf("Pod: %s\n", target.Labels["pod"])
		}
		if client.pcVerbose {
			for k, v := range target.DiscoveredLabels {
				fmt.Printf("%s: %s\n", k, v)
			}
		}
		fmt.Printf("\n")
	}

	fmt.Printf("Dropped targets:\n");
	for _, target := range result.Dropped {
		if *job != "" && *job != target.DiscoveredLabels["job"] {
			continue;
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

func main() {
	var client promClient;
	var promURL, cmd, job *string
	var verbose *bool
	var cancel context.CancelFunc

	promURL = flag.String("promurl", "", "URL of Prometheus server")
	cmd = flag.String("command", "", "<targets>")
	job = flag.String("job", "", "show only targets from specified job")
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

	apiclient, err := api.NewClient(api.Config{
		Address: *promURL,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	client.pcVerbose = *verbose;
	client.pcAPI = v1.NewAPI(apiclient)
	client.pcCtx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

	switch *cmd {
	case "targets":
		promTargets(&client, job)
	default:
		fmt.Fprintf(os.Stderr, "Invalid command: %s\n", cmd)
		os.Exit(2)
	}

	cancel()
	os.Exit(0)
}
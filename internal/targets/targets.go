package targets

import (
	"context"
	"fmt"
	"os"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type TargetArgs struct {
	Verbose bool
	// if true, only output information for active targets
	Active bool
	// if true, only output information for active targets who's state is "down"
	Down bool
	// if true, just print a count of the matching targets found
	Count bool
	// only print information on targets associated with the specified job
	Job string
}

//
// Dump information on Prometheus targets
//
func Targets(ctx context.Context, api v1.API, args *TargetArgs) {
	var count int32

	result, err := api.Targets(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of targets: %v\n", err)
		os.Exit(1)
	}

	if !args.Down && !args.Count {
		fmt.Printf("Active targets\n")
		fmt.Printf("==============\n")
	}
	for _, target := range result.Active {

		if args.Job != "" && args.Job != string(target.Labels["job"]) {
			continue
		}
		if args.Down && target.Health != v1.HealthBad {
			continue
		}

		count++
		if args.Count {
			continue
		}

		fmt.Printf("%-20s %s\n", "Scrape URL:", target.ScrapeURL)
		fmt.Printf("%-20s %s\n", "Last Scrape:", target.LastScrape.Local().String())
		fmt.Printf("%-20s %s\n", "Jobs:", target.Labels["job"])
		if target.Labels["pod"] != "" {
			fmt.Printf("%-20s %s\n", "Pod:", target.Labels["pod"])
		}
		fmt.Printf("%-20s %s\n", "State:", target.Health)
		if target.Health == v1.HealthBad {
			fmt.Printf("%-20s %s\n", "Error:", target.LastError)
		}

		if args.Verbose {
			for k, v := range target.DiscoveredLabels {
				fmt.Printf("%-20s %s\n", k, v)
			}
		}
		fmt.Printf("\n")
	}

	if args.Active {
		if args.Count {
			fmt.Printf("%d\n", count)
		}
		return
	}

	fmt.Printf("Dropped targets:\n")
	for _, target := range result.Dropped {
		if args.Job != "" && args.Job != target.DiscoveredLabels["job"] {
			continue
		}
		fmt.Printf("Job: %s\n", target.DiscoveredLabels["job"])
		if args.Verbose {
			for k, v := range target.DiscoveredLabels {
				fmt.Printf("%s: %s\n", k, v)
			}
		}
		fmt.Printf("\n")
	}
}

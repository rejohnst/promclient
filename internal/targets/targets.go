package targets

import (
	"context"
	"fmt"
	"io"
	"sort"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type TargetArgs struct {
	Verbose bool
	// Active prints only active targets.
	Active bool
	// Down prints only active targets whose state is "down".
	Down bool
	// Count prints only a count of matching active targets.
	Count bool
	// Job filters targets by job label.
	Job string
}

// Targets prints Prometheus target information.
func Targets(ctx context.Context, api v1.API, w io.Writer, args *TargetArgs) error {
	result, err := api.Targets(ctx)
	if err != nil {
		return fmt.Errorf("retrieve targets: %w", err)
	}

	count := 0
	if !args.Down && !args.Count {
		fmt.Fprintln(w, "Active targets")
		fmt.Fprintln(w, "==============")
	}

	for _, target := range result.Active {
		if !matchesActiveTarget(target, args.Job, args.Down) {
			continue
		}

		count++
		if args.Count {
			continue
		}

		fmt.Fprintf(w, "%-20s %s\n", "Scrape URL:", target.ScrapeURL)
		fmt.Fprintf(w, "%-20s %s\n", "Last Scrape:", target.LastScrape.Local().String())
		fmt.Fprintf(w, "%-20s %s\n", "Jobs:", target.Labels["job"])
		if target.Labels["pod"] != "" {
			fmt.Fprintf(w, "%-20s %s\n", "Pod:", target.Labels["pod"])
		}
		fmt.Fprintf(w, "%-20s %s\n", "State:", target.Health)
		if target.Health == v1.HealthBad {
			fmt.Fprintf(w, "%-20s %s\n", "Error:", target.LastError)
		}

		if args.Verbose {
			printStringMap(w, target.DiscoveredLabels, "%-20s %s\n")
		}
		fmt.Fprintln(w)
	}

	if args.Count {
		_, err := fmt.Fprintf(w, "%d\n", count)
		return err
	}
	if args.Active {
		return nil
	}

	fmt.Fprintln(w, "Dropped targets:")
	for _, target := range result.Dropped {
		if args.Job != "" && args.Job != target.DiscoveredLabels["job"] {
			continue
		}
		fmt.Fprintf(w, "Job: %s\n", target.DiscoveredLabels["job"])
		if args.Verbose {
			printStringMap(w, target.DiscoveredLabels, "%s: %s\n")
		}
		fmt.Fprintln(w)
	}

	return nil
}

func matchesActiveTarget(target v1.ActiveTarget, job string, downOnly bool) bool {
	if job != "" && job != string(target.Labels["job"]) {
		return false
	}
	if downOnly && target.Health != v1.HealthBad {
		return false
	}
	return true
}

func printStringMap(w io.Writer, values map[string]string, format string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Fprintf(w, format, key, values[key])
	}
}

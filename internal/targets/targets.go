package targets

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rejohnst/promclient/internal/promapi"
)

type TargetArgs struct {
	Verbose bool
	// Down prints only active targets whose state is "down".
	Down bool
	// Count prints only a count of matching active targets.
	Count bool
	// Job filters targets by job label.
	Job string
}

// Targets prints Prometheus target information. Warnings go to stderr so that
// a partial response is not mistaken for an empty one.
func Targets(ctx context.Context, client api.Client, stdout, stderr io.Writer, args *TargetArgs) error {
	buffered := bufio.NewWriter(stdout)
	err := printTargets(ctx, client, buffered, stderr, args)
	if flushErr := buffered.Flush(); err == nil {
		err = flushErr
	}
	return err
}

func printTargets(ctx context.Context, client api.Client, w, stderr io.Writer, args *TargetArgs) error {
	active, warnings, err := promapi.FetchActiveTargets(ctx, client)
	promapi.PrintWarnings(stderr, warnings)
	if err != nil {
		return err
	}

	count := 0
	if !args.Down && !args.Count {
		fmt.Fprintln(w, "Active targets")
		fmt.Fprintln(w, "==============")
	}

	for _, target := range active {
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
			printDiscoveredLabels(w, target.DiscoveredLabels)
		}
		fmt.Fprintln(w)
	}

	if args.Count {
		_, err := fmt.Fprintf(w, "%d\n", count)
		return err
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

func printDiscoveredLabels(w io.Writer, labels map[string]string) {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Fprintf(w, "%-20s %s\n", key, labels[key])
	}
}

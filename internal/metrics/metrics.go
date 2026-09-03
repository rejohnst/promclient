package metrics

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rejohnst/promclient/internal/promapi"
)

type MetricArgs struct {
	// Count prints only a count of matching metrics.
	Count bool
	// CSV prints parseable metric metadata as CSV.
	CSV bool
	// Job filters metrics by job label.
	Job string
}

// Metrics prints Prometheus metric metadata. Warnings go to stderr so that a
// partial response is not mistaken for an empty one.
func Metrics(ctx context.Context, client api.Client, w, stderr io.Writer, args *MetricArgs) error {
	metadataAPI := v1.NewAPI(client)

	jobs, err := jobs(ctx, client, stderr, args.Job)
	if err != nil {
		return err
	}

	count := 0
	var csvWriter *csv.Writer
	if args.CSV {
		csvWriter = csv.NewWriter(w)
	}

	for _, job := range jobs {
		match := fmt.Sprintf("{job=%s}", strconv.Quote(job))
		metadata, err := metadataAPI.TargetsMetadata(ctx, match, "", "")
		if err != nil {
			return fmt.Errorf("retrieve metric metadata for %s: %w", job, err)
		}

		for _, metric := range metadata {
			count++
			if args.Count {
				continue
			}

			if args.CSV {
				if err := csvWriter.Write([]string{job, metric.Metric, metric.Help, string(metric.Type)}); err != nil {
					return fmt.Errorf("write CSV row: %w", err)
				}
				continue
			}

			fmt.Fprintf(w, "%-20s %s\n", "Job:", job)
			fmt.Fprintf(w, "%-20s %s\n", "Metric Name:", metric.Metric)
			fmt.Fprintf(w, "%-20s %s\n", "Metric Help:", metric.Help)
			fmt.Fprintf(w, "%-20s %s\n", "Metric Type:", metric.Type)
			fmt.Fprintln(w)
		}
	}

	if args.CSV {
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return fmt.Errorf("flush CSV output: %w", err)
		}
	}

	if args.Count {
		_, err := fmt.Fprintf(w, "%d\n", count)
		return err
	}

	return nil
}

func jobs(ctx context.Context, client api.Client, stderr io.Writer, job string) ([]string, error) {
	if job != "" {
		return []string{job}, nil
	}

	active, warnings, err := promapi.FetchActiveTargets(ctx, client)
	promapi.PrintWarnings(stderr, warnings)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, target := range active {
		if target.Health == v1.HealthBad {
			continue
		}

		job := string(target.Labels["job"])
		if job == "" {
			continue
		}
		seen[job] = struct{}{}
	}

	jobs := make([]string, 0, len(seen))
	for job := range seen {
		jobs = append(jobs, job)
	}
	sort.Strings(jobs)

	return jobs, nil
}

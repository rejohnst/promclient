package metrics

import (
	"context"
	"fmt"
	"os"
	"strings"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type MetricArgs struct {
	Verbose bool
	// if true, just print a count of the matching metrics found
	Count bool
	// if true, print parseable outout ib CSV format
	Csv bool
	// only print information on metrics associated with the specified job
	Job string
}

func seqSearch(key string, arr []string) bool {
	for _, val := range arr {
		if val == key {
			return true
		}
	}
	return false
}

//
// Dump Prometheus' metric metadata
//
// If the -csv CLI option was set then output metric metadata in CSV format:
//
// <job>,<metric-name>,<metric-help>,<metric-type>
//
func Metrics(ctx context.Context, api v1.API, args *MetricArgs) {
	var jobs []string
	var count int32

	if args.Job != "" {
		jobs = append(jobs, args.Job)
	} else {
		result, err := api.Targets(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error retrieving list of targets: %v\n", err)
			os.Exit(1)
		}

		for _, target := range result.Active {
			// Don't add if the target is down
			if target.Health == v1.HealthBad {
				continue
			}

			// Don't add if we've already added this job to the array
			job := string(target.Labels["job"])
			if seqSearch(job, jobs) {
				continue
			}

			jobs = append(jobs, job)
		}
	}

	for _, j := range jobs {
		match := fmt.Sprintf("{job=\"%s\"}", j)
		metrics, err := api.TargetsMetadata(ctx, match, "", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error retrieving metric metadata for %s: %v\n", j, err)
			os.Exit(1)
		}
		for _, metric := range metrics {
			count++
			if args.Count {
				continue
			}

			if args.Csv {
				// Since we're outputting a CSV file, we need to replace any
				// comma chars in the help string with something else
				help := strings.ReplaceAll(metric.Help, ",", ";")
				fmt.Printf("%s,%s,%s,%s\n", j, metric.Metric, help, metric.Type)
			} else {
				fmt.Printf("%-20s %s\n", "Job:", j)
				fmt.Printf("%-20s %s\n", "Metric Name:", metric.Metric)
				fmt.Printf("%-20s %s\n", "Metric Help:", metric.Help)
				fmt.Printf("%-20s %s\n", "Metric Type:", metric.Type)
				fmt.Printf("\n")
			}
		}
	}
	if args.Count {
		fmt.Printf("%d\n", count)
	}
}

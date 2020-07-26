package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	str2duration "github.com/xhit/go-str2duration"
)

type promClient struct {
	pcAPI     v1.API
	pcCtx     context.Context
	pcVerbose bool
}

type promQuery struct {
	pqQuery		string
	pqRange		v1.Range
}

//
// Dump information on Prometheus targets
//
// args:
// job: only print information on targets associated with the specified job
// active: if true, only output information for active targets
// down: if true, only output infromation for active targets who's state is "down"
//
// returns: void
//
func promTargets(client *promClient, job string, active bool, down bool) {
	result, err := client.pcAPI.Targets(client.pcCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of targets: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Active targets\n")
	fmt.Printf("==============\n")
	for _, target := range result.Active {

		if job != "" && job != string(target.Labels["job"]) {
			continue
		}
		if down && target.Health != v1.HealthBad {
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
		if job != "" && job != target.DiscoveredLabels["job"] {
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

//
// Dump all active alerts
//
// returns: void
//
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
// args:
// job: only print metric metadata associated with the specified job
// csv: if true, output metric metadata in CSV format
//
// returns: void
//
func promMetrics(client *promClient, job string, csv bool) {
	var jobs []string

	if job != "" {
		jobs = append(jobs, job)
	} else {
		result, err := client.pcAPI.Targets(client.pcCtx)
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
		metrics, err := client.pcAPI.TargetsMetadata(client.pcCtx, match, "", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error retrieving metric metadata for %s: %v\n", j, err)
			os.Exit(1)
		}
		for _, metric := range metrics {
			if csv {
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
}

//
// Perform an instant query and print the results
//
// args:
// query: PromQL query string
//
// returns: void
//
func promInstantQuery(client *promClient, query string) {
	ts := time.Now()
	val, warnings, err := client.pcAPI.Query(client.pcCtx, query, ts)

	if err != nil {
		fmt.Fprintf(os.Stderr, "query failed: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "warnings: %v\n", warnings)
	}
	fmt.Printf("%v\n", val)
}

//
// Perform a range query and print the results
//
// args:
// pq:	pointer to promQuery struct
//
// returns: void
//
func promRangeQuery(client *promClient, pq *promQuery) {
	result, warnings, err := client.pcAPI.QueryRange(client.pcCtx, pq.pqQuery, pq.pqRange)

	if err != nil {
		fmt.Fprintf(os.Stderr, "query failed: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "warnings: %v\n", warnings)
	}
	fmt.Printf("%v\n", result)
}


func main() {
	var client promClient
	var pq promQuery
	var promURL, cmd, job, query, len, step *string
	var verbose, active, down, csv *bool
	var cancel context.CancelFunc

	promURL = flag.String("promurl", "", "URL of Prometheus server")
	cmd = flag.String("command", "", "<targets|alerts|metrics>")
	job = flag.String("job", "", "show only targets/metrics from specified job")
	query = flag.String("query", "", "PromQL query string")
	len = flag.String("len", "", "Legnth of query range")
	step = flag.String("step", "1m", "Range resolution")
	active = flag.Bool("active", false, "only display active targets")
	down = flag.Bool("down", false, "only display active targets that are down (implies -active)")
	verbose = flag.Bool("verbose", false, "enable verbose mode")
	csv = flag.Bool("csv", false, "output metric metadata as CSV")
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
	case "metrics":
		promMetrics(&client, *job, *csv)
	case "query":
		if *query == "" {
			fmt.Fprintf(os.Stderr, "-query argument is required for query command")
			os.Exit(2);
		}
		// If the len option was specifiec, we'll do a range query.  Otherwise,
		// we'll do an instant query
		if *len != "" {
			lenDur, err := str2duration.Str2Duration(*len)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to parse len: %s\n", *len)
				os.Exit(1)
			}
			stepDur, err := str2duration.Str2Duration(*step)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to parse step: %s\n", *len)
				os.Exit(1)
			}

			pq.pqQuery = *query
			pq.pqRange = v1.Range{
				Start: time.Now().Add(-lenDur),
				End: time.Now(),
				Step: stepDur,
			}

			promRangeQuery(&client, &pq)
		} else {
			promInstantQuery(&client, *query)
		}
	case "targets":
		promTargets(&client, *job, *active, *down)
	default:
		fmt.Fprintf(os.Stderr, "Invalid command: %s\n", *cmd)
		os.Exit(2)
	}

	cancel()
	os.Exit(0)
}

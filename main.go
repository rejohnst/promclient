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
	"github.com/prometheus/common/model"

	str2duration "github.com/xhit/go-str2duration"
)

var (
	BuildTime   string
	GitRevision string
)

type promClient struct {
	pcAPI     v1.API
	pcCtx     context.Context
	pcVerbose bool
	pcCount   bool
	pcCSV     bool
}

type promQueryParams struct {
	pqQuery string
	pqRange v1.Range
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
	var count int32

	result, err := client.pcAPI.Targets(client.pcCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of targets: %v\n", err)
		os.Exit(1)
	}

	if !down && !client.pcCount {
		fmt.Printf("Active targets\n")
		fmt.Printf("==============\n")
	}
	for _, target := range result.Active {

		if job != "" && job != string(target.Labels["job"]) {
			continue
		}
		if down && target.Health != v1.HealthBad {
			continue
		}

		count++
		if client.pcCount {
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
		if client.pcCount {
			fmt.Printf("%d\n", count)
		}
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
// args:
// critical: if true, only print alerts with critical severity
//
// returns: void
//
func promAlerts(client *promClient, critical bool) {
	result, err := client.pcAPI.Alerts(client.pcCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of alerts: %v\n", err)
		os.Exit(1)
	}

	if client.pcCount {
		fmt.Printf("%d\n", len(result.Alerts))
		return
	}
	fmt.Printf("\n")
	for _, alert := range result.Alerts {
		if critical && alert.Labels["severity"] != "critical" {
			continue
		}
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
//
// If the -csv CLI option was set then output metric metadata in CSV format:
//
// <job>,<metric-name>,<metric-help>,<metric-type>
//
// returns: void
//
func promMetrics(client *promClient, job string) {
	var jobs []string
	var count int32

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
			count++
			if client.pcCount {
				continue
			}

			if client.pcCSV {
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
	if client.pcCount {
		fmt.Printf("%d\n", count)
	}
}

//
// Perform a PromQL query and print the results
//
// args:
// pq:	  pointer to promQueryParam struct
// timed: if true, print wallclock time elapsed during query
//
// returns: void
//
func promQuery(client *promClient, pq *promQueryParams, timed bool) {
	var result model.Value
	var warnings v1.Warnings
	var err error
	var t1, t2 time.Time

	t1 = time.Now()
	if (v1.Range{}) == pq.pqRange {
		result, warnings, err = client.pcAPI.Query(client.pcCtx, pq.pqQuery, time.Now())
	} else {
		result, warnings, err = client.pcAPI.QueryRange(client.pcCtx, pq.pqQuery, pq.pqRange)
	}
	t2 = time.Now()

	if err != nil {
		fmt.Fprintf(os.Stderr, "query failed: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "warnings: %v\n", warnings)
	}

	//
	// result will contain a model.Value, which is a generic interface, so we
	// use a type switch to process it.  Instant queries should return type
	// model.Vector and range queries should return type model.Matrix.
	//
	switch {
	case result.Type() == model.ValVector:
		samples := result.(model.Vector)
		for _, sample := range samples {
			fmt.Printf("%s = %v @[%s]\n", sample.Metric.String(), sample.Value,
				sample.Timestamp.Time().Format("Jan 2 15:04:05 -0700 MST"))
		}
	case result.Type() == model.ValMatrix:
		streams := result.(model.Matrix)
		for _, stream := range streams {
			fmt.Printf("%s = \n", stream.Metric.String())
			for _, val := range stream.Values {
				fmt.Printf("\t%v @[%s]\n", val.Value,
					val.Timestamp.Time().Format("Jan 2 15:04:05 -0700 MST"))
			}
		}
	case result.Type() == model.ValScalar:
		fmt.Printf("%v\n", result)
	case result.Type() == model.ValString:
		fmt.Printf("%v\n", result)
	}

	if timed {
		elapsed := t2.Sub(t1)
		// time.Format is weird - see https://golang.org/pkg/time/#Time.Format
		fmt.Printf("query time: %s\n", time.Time{}.Add(elapsed).Format("04:05"))
	}
}

func main() {
	var client promClient
	var pq promQueryParams
	var promURL, cmd, job, query, len, step *string
	var verbose, active, down, csv, timed, count, version, critical *bool
	var cancel context.CancelFunc

	promURL = flag.String("promurl", "", "URL of Prometheus server")
	cmd = flag.String("command", "", "<targets|alerts|metrics>")
	job = flag.String("job", "", "show only targets/metrics from specified job")
	query = flag.String("query", "", "PromQL query string")
	version = flag.Bool("version", false, "Output program version and exit")
	len = flag.String("len", "", "Legnth of query range")
	step = flag.String("step", "1m", "Range resolution")
	timed = flag.Bool("timed", false, "Show query time")
	active = flag.Bool("active", false, "only display active targets")
	down = flag.Bool("down", false, "only display active targets that are down (implies -active)")
	count = flag.Bool("count", false, "only display a count of the requested items")
	verbose = flag.Bool("verbose", false, "enable verbose mode")
	csv = flag.Bool("csv", false, "output metric metadata as CSV")
	critical = flag.Bool("critical", false, "only show critical alerts")
	flag.Parse()

	if *version {
		fmt.Printf("Git Revision: %s\n", GitRevision)
		fmt.Printf("Build Time:   %s\n", BuildTime)
		os.Exit(0)
	}

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
	client.pcCount = *count
	client.pcCSV = *csv
	client.pcAPI = v1.NewAPI(apiclient)
	client.pcCtx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

	switch *cmd {
	case "alerts":
		promAlerts(&client, *critical)
	case "metrics":
		promMetrics(&client, *job)
	case "query":
		if *query == "" {
			fmt.Fprintf(os.Stderr, "-query argument is required for query command")
			os.Exit(2)
		}
		pq.pqQuery = *query

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

			pq.pqRange = v1.Range{
				Start: time.Now().Add(-lenDur),
				End:   time.Now(),
				Step:  stepDur,
			}
		}
		promQuery(&client, &pq, *timed)
	case "targets":
		promTargets(&client, *job, *active, *down)
	default:
		fmt.Fprintf(os.Stderr, "Invalid command: %s\n", *cmd)
		os.Exit(2)
	}

	cancel()
	os.Exit(0)
}

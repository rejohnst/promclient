package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	str2duration "github.com/xhit/go-str2duration"
)

var (
	buildTime   string
	gitRevision string
)

type promTargetArgs struct {
	verbose bool
	// if true, only output information for active targets
	active bool
	// if true, only output information for active targets who's state is "down"
	down bool
	// if true, just print a count of the matching targets found
	count bool
	// only print information on targets associated with the specified job
	job string
}

type promAlertArgs struct {
	verbose bool
	// if true, only output information on critical alerts
	critical bool
	// if true, just print a count of the matching alerts found
	count bool
}

type promMetricArgs struct {
	verbose bool
	// if true, just print a count of the matching metrics found
	count bool
	// if true, print parseable outout ib CSV format
	csv bool
	// only print information on metrics associated with the specified job
	job string
}

type promQueryArgs struct {
	// if true, print query time elapsed
	timed bool
}

type promAlert struct {
	paSeverity string
	paDescs    []string
}
type promQueryParams struct {
	pqQuery string
	pqRange v1.Range
}

//
// Dump information on Prometheus targets
//
func promTargets(ctx context.Context, api v1.API, args *promTargetArgs) {
	var count int32

	result, err := api.Targets(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of targets: %v\n", err)
		os.Exit(1)
	}

	if !args.down && !args.count {
		fmt.Printf("Active targets\n")
		fmt.Printf("==============\n")
	}
	for _, target := range result.Active {

		if args.job != "" && args.job != string(target.Labels["job"]) {
			continue
		}
		if args.down && target.Health != v1.HealthBad {
			continue
		}

		count++
		if args.count {
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

		if args.verbose {
			for k, v := range target.DiscoveredLabels {
				fmt.Printf("%-20s %s\n", k, v)
			}
		}
		fmt.Printf("\n")
	}

	if args.active {
		if args.count {
			fmt.Printf("%d\n", count)
		}
		return
	}

	fmt.Printf("Dropped targets:\n")
	for _, target := range result.Dropped {
		if args.job != "" && args.job != target.DiscoveredLabels["job"] {
			continue
		}
		fmt.Printf("Job: %s\n", target.DiscoveredLabels["job"])
		if args.verbose {
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
func promAlerts(ctx context.Context, api v1.API, args *promAlertArgs) {
	result, err := api.Alerts(ctx)
	alerts := make(map[string]*promAlert)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of alerts: %v\n", err)
		os.Exit(1)
	}

	if args.count {
		fmt.Printf("%d\n", len(result.Alerts))
		return
	}

	fmt.Printf("\n")
	for _, alert := range result.Alerts {
		if alert.Labels["alertname"] == "Watchdog" {
			continue
		}
		if args.critical && alert.Labels["severity"] != "critical" {
			continue
		}
		key := string(alert.Labels["alertname"])
		val, ok := alerts[key]
		if !ok {
			var newAlert promAlert
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
func promMetrics(ctx context.Context, api v1.API, args *promMetricArgs) {
	var jobs []string
	var count int32

	if args.job != "" {
		jobs = append(jobs, args.job)
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
			if args.count {
				continue
			}

			if args.csv {
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
	if args.count {
		fmt.Printf("%d\n", count)
	}
}

//
// Perform a PromQL query and print the results
//
func promQuery(ctx context.Context, api v1.API, args *promQueryArgs, pq *promQueryParams) {
	var result model.Value
	var warnings v1.Warnings
	var err error
	var t1, t2 time.Time

	t1 = time.Now()
	if (v1.Range{}) == pq.pqRange {
		result, warnings, err = api.Query(ctx, pq.pqQuery, time.Now())
	} else {
		result, warnings, err = api.QueryRange(ctx, pq.pqQuery, pq.pqRange)
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
			fmt.Printf("[%s] %s = %v\n",
				sample.Timestamp.Time().Format("Jan 2 15:04:05 -0700 MST"),
				sample.Metric.String(), sample.Value)
		}
	case result.Type() == model.ValMatrix:
		streams := result.(model.Matrix)
		for _, stream := range streams {
			fmt.Printf("%s = \n", stream.Metric.String())
			for _, val := range stream.Values {
				fmt.Printf("\t[%s] %v\n",
					val.Timestamp.Time().Format("Jan 2 15:04:05 -0700 MST"),
					val.Value)
			}
		}
	case result.Type() == model.ValScalar:
		fmt.Printf("%v\n", result)
	case result.Type() == model.ValString:
		fmt.Printf("%v\n", result)
	}

	if args.timed {
		elapsed := t2.Sub(t1)
		// time.Format is weird - see https://golang.org/pkg/time/#Time.Format
		fmt.Printf("query time: %s\n", time.Time{}.Add(elapsed).Format("04:05"))
	}
}

func promRuntime(ctx context.Context, api v1.API) {
	result, err := api.Runtimeinfo(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving Primetheus runtime info: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%-30s %s\n", "CWD:", result.CWD)
	fmt.Printf("%-30s %v\n", "Last Cfg Reload Successful?:", result.ReloadConfigSuccess)
	fmt.Printf("%-30s %v\n", "Last Cfg Time:", result.LastConfigTime)
	fmt.Printf("%-30s %d\n", "# of Chunks", result.ChunkCount)
	fmt.Printf("%-30s %d\n", "# of Time Series", result.TimeSeriesCount)
	fmt.Printf("%-30s %d\n", "# of Corruptions", result.CorruptionCount)
	fmt.Printf("%-30s %d\n", "# of Go Routines", result.GoroutineCount)
	fmt.Printf("%-30s %d\n", "GOMAXPROCS", result.GOMAXPROCS)
	fmt.Printf("%-30s %s\n", "GOGC", result.GOGC)
	fmt.Printf("%-30s %s\n", "GODEBUG", result.GODEBUG)
	fmt.Printf("%-30s %s\n", "Data Retention", result.StorageRetention)
}

func usage() {
	fmt.Printf("promurl -version\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=runtime [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=targets [-active|-down] [-verbose] [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=alerts [-critical] [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=metrics [-job=<arg>] [-count] [-csv] [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=query -query=<arg> [-len=<arg>] [-step=<arg>] [-timed] [-timeout=<# secs>] [-insecure]\n\n")
	flag.Usage()
}

func main() {
	var pq promQueryParams
	var cancel context.CancelFunc

	promURL := flag.String("promurl", "", "URL of Prometheus server")
	promIP := flag.String("promip", "", "IP address of Prometheus server")
	cmd := flag.String("command", "", "<targets|alerts|metrics|query|runtime>")
	job := flag.String("job", "", "show only targets/metrics from specified job")
	query := flag.String("query", "", "PromQL query string")
	version := flag.Bool("version", false, "Output program version and exit")
	len := flag.String("len", "", "Length of query range")
	step := flag.String("step", "1m", "Range resolution")
	timed := flag.Bool("timed", false, "Show query time")
	active := flag.Bool("active", false, "only display active targets")
	down := flag.Bool("down", false, "only display active targets that are down (implies -active)")
	count := flag.Bool("count", false, "only display a count of the requested items")
	verbose := flag.Bool("verbose", false, "enable verbose mode")
	csv := flag.Bool("csv", false, "output metric metadata as CSV")
	critical := flag.Bool("critical", false, "only show critical alerts")
	timeout := flag.Int("timeout", 30, "request timeout length in seconds")
	insecure := flag.Bool("insecure", false, "Skip certificate verification")
	flag.Parse()

	if *version {
		fmt.Printf("Git Revision: %s\n", gitRevision)
		fmt.Printf("Build Time:   %s\n", buildTime)
		os.Exit(0)
	}

	if *promURL == "" && *promIP == "" {
		fmt.Fprintf(os.Stderr, "Either -promurl or -promip must be specified\n\n")
		usage()
		os.Exit(2)
	}
	if *promURL != "" && *promIP != "" {
		fmt.Fprintf(os.Stderr, "-promurl and -promip are mutually exclusive\n\n")
		usage()
		os.Exit(2)
	}
	if *promIP != "" {
		*promURL = fmt.Sprintf("http://%s:9090", *promIP)
	}
	if *cmd == "" {
		fmt.Fprintf(os.Stderr, "-command is a required argument\n\n")
		usage()
		os.Exit(2)
	}
	if *down {
		*active = true
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: *insecure,
	}
	var rt http.RoundTripper = &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   5 * time.Second,
		TLSClientConfig:       tlsConfig,
		ResponseHeaderTimeout: time.Duration(time.Duration(*timeout) * time.Second),
		DisableKeepAlives:     true,
	}

	apiclient, err := api.NewClient(api.Config{
		Address:      *promURL,
		RoundTripper: rt,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	api := v1.NewAPI(apiclient)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)

	switch *cmd {
	case "alerts":
		args := promAlertArgs{*verbose, *critical, *count}
		promAlerts(ctx, api, &args)
	case "metrics":
		args := promMetricArgs{*verbose, *count, *csv, *job}
		promMetrics(ctx, api, &args)
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
		args := promQueryArgs{*timed}
		promQuery(ctx, api, &args, &pq)
	case "runtime":
		promRuntime(ctx, api)
	case "targets":
		args := promTargetArgs{*verbose, *active, *down, *count, *job}
		promTargets(ctx, api, &args)
	default:
		fmt.Fprintf(os.Stderr, "Invalid command: %s\n", *cmd)
		os.Exit(2)
	}

	cancel()
	os.Exit(0)
}

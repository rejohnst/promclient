package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rejohnst/promclient/internal/alerts"
	"github.com/rejohnst/promclient/internal/metrics"
	"github.com/rejohnst/promclient/internal/query"
	"github.com/rejohnst/promclient/internal/rules"
	"github.com/rejohnst/promclient/internal/runtime"
	"github.com/rejohnst/promclient/internal/targets"

	str2duration "github.com/xhit/go-str2duration/v2"
)

var (
	buildTime   string
	gitRevision string
)

func usage() {
	fmt.Printf("promurl -version\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=alerts [-severity=<severity>] [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=metrics [-job=<arg>] [-count] [-csv] [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=query -query=<arg> [-len=<arg>] [-step=<arg>] [-timed] [-timeout=<# secs>] [-insecure]\n\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=runtime [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=rules [-rule=<arg>|-group=<arg>] [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=targets [-active|-down] [-verbose] [-timeout=<# secs>] [-insecure]\n")
	flag.Usage()
}

func main() {
	var pq query.QueryParams
	var cancel context.CancelFunc

	// Flags common to multiple commands
	promURL := flag.String("promurl", "", "URL of Prometheus server")
	promIP := flag.String("promip", "", "IP address of Prometheus server")
	cmd := flag.String("command", "", "<targets|alerts|metrics|query|runtime>")
	job := flag.String("job", "", "show only targets/metrics from specified job")
	count := flag.Bool("count", false, "only display a count of the requested items")
	verbose := flag.Bool("verbose", false, "enable verbose mode")
	timeout := flag.Int("timeout", 30, "request timeout length in seconds")
	insecure := flag.Bool("insecure", false, "Skip TLS certificate verification")
	version := flag.Bool("version", false, "Output program version and exit")

	// Flags for query command
	promquery := flag.String("query", "", "PromQL query string")
	len := flag.String("len", "", "Length of query range")
	step := flag.String("step", "1m", "Range resolution")
	timed := flag.Bool("timed", false, "Show query time")
	skipTimestamp := flag.Bool("skip_timestamp", true, "Skip timestamp in query output")

	// Flags for target command
	active := flag.Bool("active", false, "only display active targets")
	down := flag.Bool("down", false, "only display active targets that are down (implies -active)")

	// Flags for metrics command
	csv := flag.Bool("csv", false, "output metric metadata as CSV")

	// Flags for alerts command
	severity := flag.String("severity", "", "filter alerts by specified severity")

	// Flags for rules command
	rulename := flag.String("rule", "", "Prometheus rule name")
	grpame := flag.String("group", "", "Prometheus rule group")

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
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(*timeout)*time.Second)

	switch *cmd {
	case "alerts":
		args := alerts.AlertArgs{Severity: severity, Count: *count}
		alerts.Alerts(ctx, api, &args)
	case "metrics":
		args := metrics.MetricArgs{
			Verbose: *verbose,
			Count:   *count,
			Csv:     *csv,
			Job:     *job}
		metrics.Metrics(ctx, api, &args)
	case "query":
		if *promquery == "" {
			fmt.Fprintf(os.Stderr, "-query argument is required for query command")
			os.Exit(2)
		}
		pq.Query = *promquery

		// If the len option was specifiec, we'll do a range query.  Otherwise,
		// we'll do an instant query
		if *len != "" {
			lenDur, err := str2duration.ParseDuration(*len)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to parse len: %s\n", *len)
				os.Exit(1)
			}
			stepDur, err := str2duration.ParseDuration(*step)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to parse step: %s\n", *len)
				os.Exit(1)
			}

			pq.Range = v1.Range{
				Start: time.Now().Add(-lenDur),
				End:   time.Now(),
				Step:  stepDur,
			}
		}
		args := query.QueryArgs{Timed: *timed}
		query.Query(ctx, api, &args, &pq, skipTimestamp)
	case "runtime":
		runtime.Runtime(ctx, api)
	case "rules":
		if *rulename != "" && *grpame != "" {
			fmt.Fprintf(os.Stderr, "-rule and -group are mutually exclusive")
			os.Exit(2)
		}
		args := rules.RuleArgs{RuleName: *rulename, RuleGroup: *grpame}
		rules.Rules(ctx, api, args)
	case "targets":
		args := targets.TargetArgs{
			Verbose: *verbose,
			Active:  *active,
			Down:    *down,
			Count:   *count,
			Job:     *job}
		targets.Targets(ctx, api, &args)
	default:
		fmt.Fprintf(os.Stderr, "Invalid command: %s\n", *cmd)
		os.Exit(2)
	}

	cancel()
	os.Exit(0)
}

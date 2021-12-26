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
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=alerts [-critical] [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=metrics [-job=<arg>] [-count] [-csv] [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=query -query=<arg> [-len=<arg>] [-step=<arg>] [-timed] [-timeout=<# secs>] [-insecure]\n\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=runtime [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=rules [-timeout=<# secs>] [-insecure]\n")
	fmt.Printf("promurl -promurl=<arg>|-promip=<arg> -command=targets [-active|-down] [-verbose] [-timeout=<# secs>] [-insecure]\n")
	flag.Usage()
}

func main() {
	var pq query.QueryParams
	var cancel context.CancelFunc

	promURL := flag.String("promurl", "", "URL of Prometheus server")
	promIP := flag.String("promip", "", "IP address of Prometheus server")
	cmd := flag.String("command", "", "<targets|alerts|metrics|query|runtime>")
	job := flag.String("job", "", "show only targets/metrics from specified job")
	promquery := flag.String("query", "", "PromQL query string")
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
		args := alerts.AlertArgs{*critical, *count}
		alerts.Alerts(ctx, api, &args)
	case "metrics":
		args := metrics.MetricArgs{*verbose, *count, *csv, *job}
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
		args := query.QueryArgs{*timed}
		query.Query(ctx, api, &args, &pq)
	case "runtime":
		runtime.Runtime(ctx, api)
	case "rules":
		rules.Rules(ctx, api)
	case "targets":
		args := targets.TargetArgs{*verbose, *active, *down, *count, *job}
		targets.Targets(ctx, api, &args)
	default:
		fmt.Fprintf(os.Stderr, "Invalid command: %s\n", *cmd)
		os.Exit(2)
	}

	cancel()
	os.Exit(0)
}

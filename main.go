package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
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

const defaultTimeoutSeconds = 30

var (
	buildTime   string
	gitRevision string
)

type exitError struct {
	code int
	err  error
}

func (e exitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e exitError) Unwrap() error {
	return e.err
}

func usageError(format string, args ...any) error {
	return exitError{code: 2, err: fmt.Errorf(format, args...)}
}

type clientOptions struct {
	promURL        string
	promIP         string
	timeoutSeconds int
	insecure       bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	global := clientOptions{timeoutSeconds: defaultTimeoutSeconds}
	fs := flag.NewFlagSet("promclient", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printUsage(stderr) }

	var version bool
	registerClientFlags(fs, &global)
	fs.BoolVar(&version, "version", false, "output program version and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if version {
		printVersion(stdout)
		return 0
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(stderr, "command is required")
		printUsage(stderr)
		return 2
	}

	if remaining[0] == "help" {
		err := printHelp(remaining[1:], stdout)
		if err == nil {
			return 0
		}
		var ee exitError
		if errors.As(err, &ee) {
			if ee.err != nil {
				fmt.Fprintln(stderr, ee.err)
			}
			return ee.code
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	if remaining[0] == "version" {
		printVersion(stdout)
		return 0
	}

	err := runCommand(remaining[0], remaining[1:], global, stdout, stderr)
	if err == nil {
		return 0
	}

	var ee exitError
	if errors.As(err, &ee) {
		if ee.err != nil {
			fmt.Fprintln(stderr, ee.err)
		}
		return ee.code
	}

	fmt.Fprintln(stderr, err)
	return 1
}

func runCommand(command string, args []string, global clientOptions, stdout, stderr io.Writer) error {
	switch command {
	case "alerts":
		return runAlerts(args, global, stdout, stderr)
	case "metrics":
		return runMetrics(args, global, stdout, stderr)
	case "query":
		return runQuery(args, global, stdout, stderr)
	case "runtime":
		return runRuntime(args, global, stdout, stderr)
	case "rules":
		return runRules(args, global, stdout, stderr)
	case "targets":
		return runTargets(args, global, stdout, stderr)
	default:
		return usageError("unknown command %q", command)
	}
}

func runAlerts(args []string, global clientOptions, stdout, stderr io.Writer) error {
	opts := global
	var alertArgs alerts.AlertArgs
	fs := newCommandFlagSet("alerts", stderr)
	registerClientFlags(fs, &opts)
	fs.BoolVar(&alertArgs.Count, "count", false, "only display a count of matching alerts")
	fs.StringVar(&alertArgs.Severity, "severity", "", "filter alerts by severity")

	if err := parseCommandFlags(fs, args); err != nil {
		return err
	}

	api, ctx, cancel, err := newAPI(opts)
	if err != nil {
		return err
	}
	defer cancel()

	return alerts.Alerts(ctx, api, stdout, &alertArgs)
}

func runMetrics(args []string, global clientOptions, stdout, stderr io.Writer) error {
	opts := global
	var metricArgs metrics.MetricArgs
	fs := newCommandFlagSet("metrics", stderr)
	registerClientFlags(fs, &opts)
	fs.BoolVar(&metricArgs.Count, "count", false, "only display a count of matching metrics")
	fs.BoolVar(&metricArgs.CSV, "csv", false, "output metric metadata as CSV")
	fs.StringVar(&metricArgs.Job, "job", "", "show only metrics from the specified job")

	if err := parseCommandFlags(fs, args); err != nil {
		return err
	}

	api, ctx, cancel, err := newAPI(opts)
	if err != nil {
		return err
	}
	defer cancel()

	return metrics.Metrics(ctx, api, stdout, &metricArgs)
}

func runQuery(args []string, global clientOptions, stdout, stderr io.Writer) error {
	opts := global
	var queryArgs query.QueryArgs
	var queryParams query.QueryParams
	var rangeLength string
	var step string

	fs := newCommandFlagSet("query", stderr)
	registerClientFlags(fs, &opts)
	fs.StringVar(&queryParams.Query, "query", "", "PromQL query string")
	fs.StringVar(&rangeLength, "len", "", "length of query range")
	fs.StringVar(&step, "step", "1m", "range resolution")
	fs.BoolVar(&queryArgs.Timed, "timed", false, "show query time")
	fs.BoolVar(&queryArgs.SkipTimestamp, "skip-timestamp", true, "skip timestamp in query output")
	fs.BoolVar(&queryArgs.SkipTimestamp, "skip_timestamp", true, "skip timestamp in query output")

	if err := parseCommandFlags(fs, args); err != nil {
		return err
	}
	if queryParams.Query == "" {
		return usageError("query: -query is required")
	}

	if rangeLength != "" {
		length, err := parsePositiveDuration("len", rangeLength)
		if err != nil {
			return err
		}
		stepDuration, err := parsePositiveDuration("step", step)
		if err != nil {
			return err
		}

		now := time.Now()
		queryParams.Range = v1.Range{
			Start: now.Add(-length),
			End:   now,
			Step:  stepDuration,
		}
	} else if flagWasSet(fs, "step") {
		return usageError("query: -step requires -len")
	}

	api, ctx, cancel, err := newAPI(opts)
	if err != nil {
		return err
	}
	defer cancel()

	return query.Query(ctx, api, stdout, stderr, &queryArgs, &queryParams)
}

func runRuntime(args []string, global clientOptions, stdout, stderr io.Writer) error {
	opts := global
	fs := newCommandFlagSet("runtime", stderr)
	registerClientFlags(fs, &opts)

	if err := parseCommandFlags(fs, args); err != nil {
		return err
	}

	api, ctx, cancel, err := newAPI(opts)
	if err != nil {
		return err
	}
	defer cancel()

	return runtime.Runtime(ctx, api, stdout)
}

func runRules(args []string, global clientOptions, stdout, stderr io.Writer) error {
	opts := global
	var ruleArgs rules.RuleArgs
	fs := newCommandFlagSet("rules", stderr)
	registerClientFlags(fs, &opts)
	fs.StringVar(&ruleArgs.RuleName, "rule", "", "Prometheus rule name")
	fs.StringVar(&ruleArgs.RuleGroup, "group", "", "Prometheus rule group")

	if err := parseCommandFlags(fs, args); err != nil {
		return err
	}
	if ruleArgs.RuleName != "" && ruleArgs.RuleGroup != "" {
		return usageError("rules: -rule and -group are mutually exclusive")
	}

	api, ctx, cancel, err := newAPI(opts)
	if err != nil {
		return err
	}
	defer cancel()

	return rules.Rules(ctx, api, stdout, ruleArgs)
}

func runTargets(args []string, global clientOptions, stdout, stderr io.Writer) error {
	opts := global
	var targetArgs targets.TargetArgs
	fs := newCommandFlagSet("targets", stderr)
	registerClientFlags(fs, &opts)
	fs.BoolVar(&targetArgs.Active, "active", false, "only display active targets")
	fs.BoolVar(&targetArgs.Count, "count", false, "only display a count of matching active targets")
	fs.BoolVar(&targetArgs.Down, "down", false, "only display active targets that are down")
	fs.StringVar(&targetArgs.Job, "job", "", "show only targets from the specified job")
	fs.BoolVar(&targetArgs.Verbose, "verbose", false, "enable verbose output")

	if err := parseCommandFlags(fs, args); err != nil {
		return err
	}
	if targetArgs.Down {
		targetArgs.Active = true
	}

	api, ctx, cancel, err := newAPI(opts)
	if err != nil {
		return err
	}
	defer cancel()

	return targets.Targets(ctx, api, stdout, &targetArgs)
}

func newCommandFlagSet(command string, output io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("promclient "+command, flag.ContinueOnError)
	fs.SetOutput(output)
	return fs
}

func registerClientFlags(fs *flag.FlagSet, opts *clientOptions) {
	fs.StringVar(&opts.promURL, "promurl", opts.promURL, "URL of Prometheus server")
	fs.StringVar(&opts.promIP, "promip", opts.promIP, "IP address of Prometheus server")
	fs.IntVar(&opts.timeoutSeconds, "timeout", opts.timeoutSeconds, "request timeout length in seconds")
	fs.BoolVar(&opts.insecure, "insecure", opts.insecure, "skip TLS certificate verification")
}

func parseCommandFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitError{code: 0}
		}
		return exitError{code: 2}
	}
	if fs.NArg() != 0 {
		return usageError("%s: unexpected argument(s): %s", fs.Name(), strings.Join(fs.Args(), " "))
	}
	return nil
}

func newAPI(opts clientOptions) (v1.API, context.Context, context.CancelFunc, error) {
	address, timeout, err := opts.addressAndTimeout()
	if err != nil {
		return nil, nil, nil, err
	}

	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, nil, nil, fmt.Errorf("default transport has unexpected type %T", http.DefaultTransport)
	}
	transport := baseTransport.Clone()
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = dialer.DialContext
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.ResponseHeaderTimeout = timeout
	transport.DisableKeepAlives = true

	if opts.insecure {
		tlsConfig := &tls.Config{}
		if transport.TLSClientConfig != nil {
			tlsConfig = transport.TLSClientConfig.Clone()
		}
		tlsConfig.InsecureSkipVerify = true
		transport.TLSClientConfig = tlsConfig
	}

	client, err := api.NewClient(api.Config{
		Address:      address,
		RoundTripper: transport,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create Prometheus client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return v1.NewAPI(client), ctx, cancel, nil
}

func (opts clientOptions) addressAndTimeout() (string, time.Duration, error) {
	if opts.promURL == "" && opts.promIP == "" {
		return "", 0, usageError("either -promurl or -promip must be specified")
	}
	if opts.promURL != "" && opts.promIP != "" {
		return "", 0, usageError("-promurl and -promip are mutually exclusive")
	}
	if opts.timeoutSeconds <= 0 {
		return "", 0, usageError("-timeout must be greater than zero")
	}

	address := opts.promURL
	if opts.promIP != "" {
		address = "http://" + net.JoinHostPort(opts.promIP, "9090")
	}

	return address, time.Duration(opts.timeoutSeconds) * time.Second, nil
}

func parsePositiveDuration(flagName, value string) (time.Duration, error) {
	duration, err := str2duration.ParseDuration(value)
	if err != nil {
		return 0, usageError("failed to parse -%s: %s", flagName, value)
	}
	if duration <= 0 {
		return 0, usageError("-%s must be greater than zero", flagName)
	}
	return duration, nil
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "Git Revision: %s\n", gitRevision)
	fmt.Fprintf(w, "Build Time:   %s\n", buildTime)
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  promclient -version")
	fmt.Fprintln(w, "  promclient [client flags] <command> [command flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  alerts    Print active alerts")
	fmt.Fprintln(w, "  metrics   Print metric metadata")
	fmt.Fprintln(w, "  query     Query the Prometheus TSDB")
	fmt.Fprintln(w, "  runtime   Print Prometheus runtime information")
	fmt.Fprintln(w, "  rules     Print Prometheus rule status")
	fmt.Fprintln(w, "  targets   Print target status")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Client flags may be set before or after the command.")
	fmt.Fprintln(w, `Use "promclient <command> -h" for command-specific flags.`)
}

func printHelp(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}
	if len(args) > 1 {
		return usageError("help accepts at most one command name")
	}

	fs := newCommandFlagSet(args[0], stdout)
	opts := clientOptions{timeoutSeconds: defaultTimeoutSeconds}
	registerClientFlags(fs, &opts)

	switch args[0] {
	case "alerts":
		var commandArgs alerts.AlertArgs
		fs.BoolVar(&commandArgs.Count, "count", false, "only display a count of matching alerts")
		fs.StringVar(&commandArgs.Severity, "severity", "", "filter alerts by severity")
	case "metrics":
		var commandArgs metrics.MetricArgs
		fs.BoolVar(&commandArgs.Count, "count", false, "only display a count of matching metrics")
		fs.BoolVar(&commandArgs.CSV, "csv", false, "output metric metadata as CSV")
		fs.StringVar(&commandArgs.Job, "job", "", "show only metrics from the specified job")
	case "query":
		var commandArgs query.QueryArgs
		fs.String("query", "", "PromQL query string")
		fs.String("len", "", "length of query range")
		fs.String("step", "1m", "range resolution")
		fs.BoolVar(&commandArgs.Timed, "timed", false, "show query time")
		fs.BoolVar(&commandArgs.SkipTimestamp, "skip-timestamp", true, "skip timestamp in query output")
	case "runtime":
	case "rules":
		var commandArgs rules.RuleArgs
		fs.StringVar(&commandArgs.RuleName, "rule", "", "Prometheus rule name")
		fs.StringVar(&commandArgs.RuleGroup, "group", "", "Prometheus rule group")
	case "targets":
		var commandArgs targets.TargetArgs
		fs.BoolVar(&commandArgs.Active, "active", false, "only display active targets")
		fs.BoolVar(&commandArgs.Count, "count", false, "only display a count of matching active targets")
		fs.BoolVar(&commandArgs.Down, "down", false, "only display active targets that are down")
		fs.StringVar(&commandArgs.Job, "job", "", "show only targets from the specified job")
		fs.BoolVar(&commandArgs.Verbose, "verbose", false, "enable verbose output")
	default:
		return usageError("unknown command %q", args[0])
	}

	fs.PrintDefaults()
	return nil
}

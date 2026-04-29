# promclient

CLI for the Prometheus HTTP API.

## Build and Install

```sh
go install github.com/rejohnst/promclient@latest
```

For a local checkout:

```sh
make build
make install
```

## Usage

```sh
promclient -version
promclient [client flags] <command> [command flags]
```

Client flags may be set before or after the command:

```text
  -insecure
        skip TLS certificate verification
  -promip string
        IP address of Prometheus server; uses http://<ip>:9090
  -promurl string
        URL of Prometheus server
  -timeout int
        request timeout length in seconds (default 30)
```

Exactly one of `-promurl` or `-promip` is required for commands that contact Prometheus.

## Commands

### Runtime

```sh
promclient runtime -promurl=<url>
```

Print Prometheus runtime information.

### Targets

```sh
promclient targets -promurl=<url> [-active|-down] [-job=<job>] [-count] [-verbose]
```

Print target status. `-down` implies `-active`. `-count` prints the count of matching active targets.

### Alerts

```sh
promclient alerts -promurl=<url> [-severity=<severity>] [-count]
```

Print firing alerts, excluding Watchdog. `-count` applies the same filtering as normal alert output.

### Metrics

```sh
promclient metrics -promurl=<url> [-job=<job>] [-count] [-csv]
```

Print target metric metadata. `-csv` emits valid CSV with fields:

```text
job,metric-name,metric-help,metric-type
```

### Rules

```sh
promclient rules -promurl=<url> [-rule=<name>|-group=<name>]
```

Print Prometheus alerting and recording rule status.

### Query

```sh
promclient query -promurl=<url> -query=<promql> [-len=<duration>] [-step=<duration>] [-timed] [-skip-timestamp=false]
```

Without `-len`, `query` runs an instant query. With `-len`, it runs a range query ending at the current time. `-step` requires `-len`.

## Examples

Print down `node-exporter` targets:

```sh
promclient targets -promurl=<url> -job=node-exporter -down
```

Show critical firing alerts:

```sh
promclient alerts -promurl=<url> -severity=critical
```

Print metric metadata for `kube-state-metrics` as CSV:

```sh
promclient metrics -promurl=<url> -job=kube-state-metrics -csv | sort -u
```

Run an instant PromQL query:

```sh
promclient query -promurl=<url> -query='up{job="node-exporter"}'
```

# promclient
CLI for Prometheus API

## Local Installation

```
go get -d github.com/rejohnst/promclient/cmd
go build -o $GOPATH/bin/promclient github.com/rejohnst/promclient/cmd
```

## Usage Summary

### Print CLI version
```
promurl -version
```

### Print Prometheus runtime configuration
```
promurl -promurl=<arg>|-promip=<arg> -command=runtime [-timeout=<# secs>] \
  [-insecure]
```

### Print Status of Prometheus Metric Targets
```
promurl -promurl=<arg>|-promip=<arg> -command=targets [-active|-down] \
  [-verbose] [-timeout=<# secs>] [-insecure]

  -active
    	only display active targets

  -down
    	only display active targets that are down (implies -active)

  -job string
    	show only targets from specified job (implies -active)
```

### Print Active Alerts
```
promurl -promurl=<arg>|-promip=<arg> -command=alerts [-critical] \
  [-timeout=<# secs>] [-insecure]

  -critical
    	only show critical alerts
```

### Print Metric Metadata
```
promurl -promurl=<arg>|-promip=<arg> -command=metrics [-job=<arg>] [-count] \
  [-csv] [-timeout=<# secs>] [-insecure]

  -csv
    	output metric metadata as CSV

  -job string
    	show only metrics from specified job
```

### Query the Prometheus TSDB
```
promurl -promurl=<arg>|-promip=<arg> -command=query -query=<arg> \
  [-len=<arg>] [-step=<arg>] [-timed] [-timeout=<# secs>] [-insecure]

  -len string
    	Length of query range

  -query string
    	PromQL query string

  -step string
    	Range resolution (default "1m")

  -timed
      Show query time
```

### CLI options common to multiple commands:
```
  -command string
    	<alerts|metrics|query|rules|runtime|targets>

  -count
    	only display a count of the requested items

  -insecure
      Skip certificate verification

  -promip string
    	IP address of Prometheus server
 
  -promurl string
    	URL of Prometheus server

  -timeout int
      Request timeout length in seconds (default 30)

  -verbose
    	enable verbose mode
```

## Example Usages

### Get Prometheus' Runtime Info

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP> -command=runtime```

### Perform an query

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP>  -command=query -query="<PromQL query>" [-len=<duration>] [-step=<duration>]```

### Print info on all Prometheus targets

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP> -command=targets```

### Print info on only active Prometheus targets

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP> -command=targets -active```

### Print info on active Prometheus targets associated with a given job

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP> -command=targets -job=<job-name>```

### Print info on active Prometheus alerts

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP> -command=alerts```

### Show only critical active Prometheus alerts

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP> -command=alerts -critical```

### Print metadata on all available metrics

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP> -command=metrics```

### Print metadata for available metrics associated with a given job

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP> -command=metrics -job=<job-name>```

### Output metadata on all available metrics in CSV format

```promclient -promurl=<Prometheus URL>|-promip=<Prometheus IP> -command=metrics -csv | sort -u```


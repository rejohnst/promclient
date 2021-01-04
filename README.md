# promclient
CLI for Prometheus API

## Usage Summary

```
promurl -version
promurl -promurl=<arg>|-promip=<arg> -command=runtime [-timeout=<# secs>]
promurl -promurl=<arg>|-promip=<arg> -command=targets [-active|-down] [-verbose] [-timeout=<# secs>]
promurl -promurl=<arg>|-promip=<arg> -command=alerts [-critical] [-timeout=<# secs>]
promurl -promurl=<arg>|-promip=<arg> -command=metrics [-job=<arg>] [-count] [-csv] [-timeout=<# secs>]
promurl -promurl=<arg>|-promip=<arg> -command=query -query=<arg> [-len=<arg>] [-step=<arg>] [-timed] [-timeout=<# secs>]

Usage of ./promclient:
  -active
    	only display active targets
  -command string
    	<targets|alerts|metrics|query|runtime>
  -count
    	only display a count of the requested items
  -critical
    	only show critical alerts
  -csv
    	output metric metadata as CSV
  -down
    	only display active targets that are down (implies -active)
  -job string
    	show only targets/metrics from specified job
  -len string
    	Length of query range
  -promip string
    	IP address of Prometheus server
  -promurl string
    	URL of Prometheus server
  -query string
    	PromQL query string
  -step string
    	Range resolution (default "1m")
  -timed
    	Show query time
  -timeout int
    	request timeout length in seconds (default 10)
  -verbose
    	enable verbose mode
  -version
    	Output program version and exit

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


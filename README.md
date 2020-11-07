# promclient
CLI for Prometheus API

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


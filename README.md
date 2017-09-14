# Monit Exporter for Prometheus

Simple server that periodically scrapes monit status and exports checks information via HTTP for Prometheus.

Build it:
```bash
go build
```

Run it:

```bash
./monit_exporter [flags]
```

## Configuration


Parameter | Description | Default
--- | --- | ---
`web.listen-address` | address and port to bind | localhost:9388
`web.metrics-path` | relative path to expose metrics | /metrics
`monit.scrape-uri` | uri to get monit status | http://localhost:2812/_status?format=xml&level=full
`monit.timeout` | timeout for getting monit status | 5s

If you need to use basic auth, use format "http://user:password@host/_status?format=xml&level=full"

# Monit Exporter for Prometheus

Simple server that periodically scrapes monit status and exports checks information via HTTP for Prometheus.

Build it:
```bash
go build
```

Run it:

```bash
./monit_exporter
```

## Configuration

The application will look for configuration in "config.toml" file located in the same directory. Use -conf flag to override config file name and location.

Configuration parameters:

Parameter | Description | Default
--- | --- | ---
`listen_address` | address and port to bind | localhost:9388
`metrics_path` | relative path to expose metrics | /metrics
`ignore_ssl` | whether of not to ignore ssl errors | false
`monit_scrape_uri` | uri to get monit status | http://localhost:2812/_status?format=xml&level=full
`monit_user` | user for monit basic auth, if needed | none
`monit_password` | password for monit status, if needed | none


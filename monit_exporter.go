package main

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
	"golang.org/x/net/html/charset"
)

const (
	namespace = "monit" // Prefix for Prometheus metrics.
)

var (
	listeningAddress = flag.String("telemetry.address", ":9388", "Address on which to expose metrics.")
	metricsEndpoint  = flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics.")
	monitScrapeURI   = flag.String("monit.scrape_uri", "http://localhost:2812/_status?format=xml&level=full", "URI to monit status page")
	insecure         = flag.Bool("insecure", true, "Ignore server certificate if using https")
)

var serviceTypes = map[int]string{
	0: "filesystem",
	2: "file",
	3: "program with pidfile",
	5: "system",
	7: "program with path",
}

type monitXML struct {
	MonitServices []monitService `xml:"service"`
}

// Simplified structure of monit check.
type monitService struct {
	Type      int    `xml:"type,attr"`
	Name      string `xml:"name"`
	Status    int    `xml:"status"`
	Monitored string `xml:"monitor"`
}

// Exporter collects monit stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	URI    string
	mutex  sync.RWMutex
	client *http.Client

	scrapeFailures prometheus.Counter
	checkStatus    *prometheus.GaugeVec
}

// Returns an initialized Exporter.
func NewExporter(uri string) *Exporter {
	return &Exporter{
		URI: uri,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrape_failures_total",
			Help:      "Number of errors while scraping monit status.",
		}),
		checkStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "service_check",
			Help:      "Monit service check info",
		},
			[]string{"check_name", "type", "monitored"},
		),
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecure},
			},
		},
	}
}

// Describe describes all the metrics ever exported by the monit exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.scrapeFailures.Describe(ch)
	e.checkStatus.Describe(ch)
}

func (e *Exporter) collect(ch chan<- prometheus.Metric) error {
	resp, err := e.client.Get(e.URI)
	if err != nil {
		return fmt.Errorf("Error scraping monit: %v", err)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if err != nil {
			data = []byte(err.Error())
		}
		return fmt.Errorf("Status %s (%d): %s", resp.Status, resp.StatusCode, data)
	}

	var response monitXML
	reader := bytes.NewReader(data)
	decoder := xml.NewDecoder(reader)

	// Parsing status results to structure
	decoder.CharsetReader = charset.NewReaderLabel
	err = decoder.Decode(&response)

	if err != nil {
		fmt.Printf("Error parsing xml response: %v", err)
	}

	// Constructing metrics
	for _, service := range response.MonitServices {
		e.checkStatus.With(prometheus.Labels{"check_name": service.Name, "type": serviceTypes[service.Type], "monitored": service.Monitored}).Set(float64(service.Status))
	}

	return nil
}

// Collect fetches the stats from configured monit location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // Protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		log.Printf("Error scraping monit: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	spew.Dump(e)
	e.checkStatus.Collect(ch)
	return
}

func main() {
	flag.Parse()

	exporter := NewExporter(*monitScrapeURI)
	prometheus.MustRegister(exporter)

	log.Printf("Starting Server: %s", *listeningAddress)
	http.Handle(*metricsEndpoint, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>Monit Exporter</title></head>
            <body>
            <h1>Monit Exporter</h1>
            <p><a href="` + *metricsEndpoint + `">Metrics</a></p>
            </body>
            </html>`))
	})

	log.Fatal(http.ListenAndServe(*listeningAddress, nil))
}

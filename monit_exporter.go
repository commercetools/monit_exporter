package main

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"flag"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
	"github.com/spf13/viper"
	"golang.org/x/net/html/charset"
)

const (
	namespace = "monit" // Prefix for Prometheus metrics.
)

var configFile = flag.String("conf", "./config.toml", "Configuration file for exporter")

var serviceTypes = map[int]string{
	0: "filesystem",
	2: "file",
	3: "program with pidfile",
	5: "system",
	7: "program with path",
}

var response monitXML

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

	up          prometheus.Gauge
	checkStatus *prometheus.GaugeVec
}

type Config struct {
	listen_address   string
	metrics_path     string
	ignore_ssl       bool
	monit_scrape_uri string
	monit_user       string
	monit_password   string
}

var MonitConfig *Config

func FetchMonitStatus(uri string) ([]byte, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: MonitConfig.ignore_ssl},
		},
	}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		log.Errorf("Unable to create request: %v", err)
	}

	req.SetBasicAuth(MonitConfig.monit_user, MonitConfig.monit_password)
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Unable to fetch monit status")
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Unable to read monit status")
		return nil, err
	}
	defer resp.Body.Close()
	return data, nil
}

func ParseMonitStatus(data []byte) error {
	reader := bytes.NewReader(data)
	decoder := xml.NewDecoder(reader)

	// Parsing status results to structure
	decoder.CharsetReader = charset.NewReaderLabel
	err := decoder.Decode(&response)

	return err
}

// Returns an initialized Exporter.
func NewExporter(uri string) (*Exporter, error) {

	return &Exporter{
		URI: uri,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "exporter_up",
			Help:      "Monit status availability",
		}),
		checkStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "exporter_service_check",
			Help:      "Monit service check info",
		},
			[]string{"check_name", "type", "monitored"},
		),
	}, nil
}

// Describe describes all the metrics ever exported by the monit exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
	e.checkStatus.Describe(ch)
}

func (e *Exporter) scrape() error {
	data, err := FetchMonitStatus(e.URI)
	if err != nil {
		// set "monit_exporter_up" gauge to 0, remove previous metrics from e.checkStatus vector
		e.up.Set(0)
		e.checkStatus.Reset()
		log.Errorf("Error getting monit status: %v", err)
		return err
	} else {
		err = ParseMonitStatus(data)
		if err != nil {
			e.up.Set(0)
			e.checkStatus.Reset()
			log.Errorf("Error parsing data from monit: %v", err)
		} else {
			e.up.Set(1)
			// Constructing metrics
			for _, service := range response.MonitServices {
				e.checkStatus.With(prometheus.Labels{"check_name": service.Name, "type": serviceTypes[service.Type], "monitored": service.Monitored}).Set(float64(service.Status))
			}
		}
		return err
	}
}

// Collect fetches the stats from configured monit location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // Protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	e.scrape()
	e.up.Collect(ch)
	e.checkStatus.Collect(ch)
	return
}

func main() {
	flag.Parse()

	v := viper.New()
	v.SetDefault("listen_address", "localhost:9388")
	v.SetDefault("metrics_path", "/metrics")
	v.SetDefault("ignore_ssl", false)
	v.SetDefault("monit_scrape_uri", "http://localhost:2812/_status?format=xml&level=full")
	v.SetDefault("monit_user", "")
	v.SetDefault("monit_password", "")
	v.SetConfigFile(*configFile)
	v.SetConfigType("toml")
	err := v.ReadInConfig() // Find and read the config file
	if err != nil {         // Handle errors reading the config file
		log.Info("Error reading config file: %s. Using defaults.", err)
	}

	MonitConfig = &Config{
		listen_address:   v.GetString("listen_address"),
		metrics_path:     v.GetString("metrics_path"),
		ignore_ssl:       v.GetBool("ignore_ssl"),
		monit_scrape_uri: v.GetString("monit_scrape_uri"),
		monit_user:       v.GetString("monit_user"),
		monit_password:   v.GetString("monit_password"),
	}

	exporter, err := NewExporter(MonitConfig.monit_scrape_uri)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)

	log.Printf("Starting monit_exporter: %s", MonitConfig.listen_address)
	http.Handle(MonitConfig.metrics_path, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>Monit Exporter</title></head>
            <body>
            <h1>Monit Exporter</h1>
            <p><a href="` + MonitConfig.metrics_path + `">Metrics</a></p>
            </body>
            </html>`))
	})

	log.Fatal(http.ListenAndServe(MonitConfig.listen_address, nil))
}

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"fmt"
	"io/ioutil"
	"time"
)

const (
	monitStatus = `<?xml version="1.0" encoding="ISO-8859-1"?><monit><server><id>acfbb9e9118e68d3754761a79d3aae16</id><incarnation>1504605214</incarnation><version>5.23.0</version><uptime>136736</uptime><poll>60</poll><startdelay>0</startdelay><localhostname>fc566edc8b68</localhostname><controlfile>/opt/monit/etc/monitrc</controlfile><httpd><address>172.17.0.2</address><port>2812</port><ssl>0</ssl></httpd></server><platform><name>Linux</name><release>4.9.27-moby</release><version>#1 SMP Thu May 11 04:01:18 UTC 2017</version><machine>x86_64</machine><cpu>4</cpu><memory>2046768</memory><swap>1048572</swap></platform><service type="5"><name>fc566edc8b68</name><collected_sec>1505209672</collected_sec><collected_usec>23215</collected_usec><status>0</status><status_hint>0</status_hint><monitor>1</monitor><monitormode>0</monitormode><onreboot>0</onreboot><pendingaction>0</pendingaction><system><load><avg01>0.00</avg01><avg05>0.00</avg05><avg15>0.00</avg15></load><cpu><user>0.1</user><system>0.1</system><wait>0.1</wait></cpu><memory><percent>6.5</percent><kilobyte>133628</kilobyte></memory><swap><percent>0.0</percent><kilobyte>0</kilobyte></swap></system></service></monit>`
)

func TestMonitStatus(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(monitStatus))
	})
	server := httptest.NewServer(handler)
	config := ParseConfig()
	config.listen_address = server.URL
	e, err := NewExporter(config)
	if err != nil {
		t.Error("Unexpected error during exporter creation")
	}
	err = e.scrape()
	if err != nil {
		t.Error("Unexpected execution error:", err)
	}
}

func TestFieldsParsing(t *testing.T) {
	parsedData, err := ParseMonitStatus([]byte(monitStatus))
	if err != nil {
		t.Error("Unable to parse XML:", err)
	}
	t.Log(parsedData)
}

func TestMonitUnavailable(t *testing.T) {
	mConfig := &Config{
		listen_address: "http://localhost:93",
		metrics_path: "/metrics",
		ignore_ssl: true,
		monit_scrape_uri: "http://localhost:2812/status",
		monit_user: "user",
		monit_password: "password",
	}
	e, err := NewExporter(mConfig)
	if err != nil {
		t.Error("Unexpected error during exporter creation")
	}
	err = e.scrape()
	if err == nil {
		t.Error("Unexpected succsessful execution")
	}
}

func TestHttpQueryExporter(t *testing.T) {
	go main()
	time.Sleep(50 * time.Millisecond)
	address := "127.0.0.1:9388"
	resp, err := http.Get(fmt.Sprintf("http://%s/metrics", address))
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Error(err)
	}
	if want, have := http.StatusOK, resp.StatusCode; want != have {
		t.Errorf("want /metrics status code %d, have %d. Body:\n%s", want, have, b)
	}

}

func TestBasicAuth(t *testing.T) {

}

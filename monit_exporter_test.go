package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	monitStatus = `<?xml version="1.0" encoding="ISO-8859-1"?><monit><server><id>acfbb9e9118e68d3754761a79d3aae16</id><incarnation>1504605214</incarnation><version>5.23.0</version><uptime>136736</uptime><poll>60</poll><startdelay>0</startdelay><localhostname>fc566edc8b68</localhostname><controlfile>/opt/monit/etc/monitrc</controlfile><httpd><address>172.17.0.2</address><port>2812</port><ssl>0</ssl></httpd></server><platform><name>Linux</name><release>4.9.27-moby</release><version>#1 SMP Thu May 11 04:01:18 UTC 2017</version><machine>x86_64</machine><cpu>4</cpu><memory>2046768</memory><swap>1048572</swap></platform><service type="5"><name>fc566edc8b68</name><collected_sec>1505209672</collected_sec><collected_usec>23215</collected_usec><status>0</status><status_hint>0</status_hint><monitor>1</monitor><monitormode>0</monitormode><onreboot>0</onreboot><pendingaction>0</pendingaction><system><load><avg01>0.00</avg01><avg05>0.00</avg05><avg15>0.00</avg15></load><cpu><user>0.1</user><system>0.1</system><wait>0.1</wait></cpu><memory><percent>6.5</percent><kilobyte>133628</kilobyte></memory><swap><percent>0.0</percent><kilobyte>0</kilobyte></swap></system></service></monit>`
	metricCount = 1
)

func TestMonitStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(monitStatus))
	})
	server := httptest.NewServer(handler)

	e := NewExporter(server.URL)
	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		e.Collect(ch)
	}()

	for i := 1; i <= metricCount; i++ {
		m := <-ch
		if m == nil {
			t.Error("expected metric but got nil")
		}
	}
	if <-ch != nil {
		t.Error("expected closed channel")
	}
}

func TestFieldsParsing(t *testing.T) {

}

func TestMonitUnavailable(t *testing) {

}

func TestBasicAuth(t *testing) {

}

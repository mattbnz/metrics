// Collects website metrics and exports to prometheus
//
// Copyright © 2023 Matt Brown.
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"mattb.nz/web/metrics/metrics"
	"mattb.nz/web/metrics/prom"
)

type MonitoredSite struct {
	Host            string
	AllowedReferers []string
}

type Config []MonitoredSite

var config Config

func getHostForReferer(referer string) string {
	for _, site := range config {
		for _, allowed := range site.AllowedReferers {
			if allowed == referer {
				return site.Host
			}
		}
	}
	return ""
}

func isKnownHost(host string) bool {
	for _, site := range config {
		if site.Host == host {
			return true
		}
	}
	return false
}

func CollectMetric(w http.ResponseWriter, r *http.Request) {
	referer := r.Header.Get("Referer")
	host := getHostForReferer(referer)
	if host == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("unknown host"))
		log.Printf("Ignoring request from unknown referer: %s", referer)
		return
	}

	// Unmarshal the request body into our Event struct.
	event := metrics.Event{}
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
		return
	}
	if !metrics.IsKnownEvent(event.Event) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("unknown event type"))
		return
	}
	if event.Event == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("no event type"))
		return
	}

	sitedata := metrics.GetSiteData(referer)
	sitedata.EventCount[event.Event]++

	w.WriteHeader(http.StatusOK)
}

// Load config from JSON file
func loadConfig(filename string) (Config, error) {
	// Open the file.
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read the contents of the file into a byte array.
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON data into the provided interface.
	config := Config{}
	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func setupHandlers() {
	// register a prometheus metric exporter
	collector := prom.Collector{}
	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", CollectMetric)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("all good"))
	})
}

func main() {
	var err error
	config, err = loadConfig(os.Getenv("CONFIG_FILE"))
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}

	setupHandlers()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

}

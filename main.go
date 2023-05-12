// Collects website metrics and exports to prometheus
//
// Copyright © 2023 Matt Brown.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"mattb.nz/web/metrics/config"
	"mattb.nz/web/metrics/db"
	"mattb.nz/web/metrics/metrics"
	"mattb.nz/web/metrics/prom"
	"mattb.nz/web/metrics/tailscale"
)

var conf config.Config

func writeCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Add("Access-Control-Allow-Origin", origin)
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	headers := r.Header.Get("Access-Control-Request-Headers")
	if headers == "" {
		headers = "*"
	}
	w.Header().Add("Access-Control-Allow-Headers", headers)

}
func CollectMetric(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	host := conf.GetHostForOrigin(origin)
	if host == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("unknown host"))
		log.Printf("Ignoring request from unknown origin: %s", origin)
		return
	}

	if r.Method == "OPTIONS" {
		writeCORSHeaders(w, r)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Unmarshal the request body into our Event struct.
	event := metrics.JsonEvent{}
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("could not decode request body"))
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

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = origin
	}
	ip := r.Header.Get("Fly-Client-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}
	logEvent := db.EventLog{
		When:     time.Now(),
		Host:     host,
		Referer:  referer,
		IP:       ip,
		RawEvent: event,
	}
	if err := db.DB.Create(&logEvent).Error; err != nil {
		log.Printf("Could not log raw event: %v", err)
	}
	sitedata := metrics.GetSiteData(host)
	sitedata.EventCount[event.Event]++

	writeCORSHeaders(w, r)
	w.WriteHeader(http.StatusOK)
}

func setupHandlers(mux *http.ServeMux) {
	// register a prometheus metric exporter
	collector := prom.Collector{}
	prometheus.MustRegister(collector)
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/", CollectMetric)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("all good"))
	})
}

func envName() string {
	env := os.Getenv("METRICS_ENV")
	if env == "" {
		env = "dev"
	}
	return env
}

func main() {
	var err error
	conf, err = config.LoadConfig(os.Getenv("CONFIG_FILE"))
	if err != nil {
		log.Fatalf("could not load config: %v", err)
	}
	if err := db.Init(conf); err != nil {
		log.Printf("No DB available, will continue with Prometheus exports only!: %v", err)
	}

	err = tailscale.Init(fmt.Sprintf("metrics-%s", envName()), conf.StateDirectory, 30*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to tailscale: %v", err)
	}

	setupHandlers(http.DefaultServeMux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Always listen on localhost
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		log.Println("listening on", port)
		http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
		wg.Done()
	}()

	// Try and also listen on TS (for admin UI)
	wg.Add(1)
	go func() {
		tailscale.Serve(http.DefaultServeMux)
		wg.Done()
	}()

	wg.Wait()
}

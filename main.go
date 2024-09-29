// Collects website metrics and exports to prometheus
//
// Copyright © 2023 Matt Brown.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"mattb.nz/web/metrics/config"
	"mattb.nz/web/metrics/db"
	"mattb.nz/web/metrics/js"
	"mattb.nz/web/metrics/metrics"
	"mattb.nz/web/metrics/prom"
	"mattb.nz/web/metrics/reporting"
	"mattb.nz/web/metrics/tailscale"
	"mattb.nz/web/metrics/templates"
)

var conf config.Config

// write CORS headers for a request
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

// wrap a handler (e.g. fs.FileServer) to add CORS headers
func serveWithCORS(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeCORSHeaders(w, r)
		h.ServeHTTP(w, r)
	}
}

// returns the request IP
func requestIP(r *http.Request) string {
	ip := r.Header.Get("Fly-Client-IP")
	if ip == "" {
		var err error
		ip, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = ""
		}
	}
	return ip
}

// returns the known origin and host if the request should continue, or empty strings in failure cases.
//
// Failure cases include either a request from an unknown origin, OR a pre-flight request
// from a known origin, in which case the pre-flight response has already been sent so further
// processing should not continue.
func checkOriginCORS(w http.ResponseWriter, r *http.Request) (string, string) {
	origin := r.Header.Get("Origin")
	host := conf.GetHostForOrigin(origin)
	if host == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("unknown host"))
		log.Printf("Ignoring request from unknown origin: %s", origin)
		return "", ""
	}

	if r.Method == "OPTIONS" {
		writeCORSHeaders(w, r)
		w.WriteHeader(http.StatusOK)
		return "", ""
	}

	return origin, host
}

type contactData struct {
	Name    string
	Org     string
	Details string
	Msg     string
}

func ContactForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	origin, host := checkOriginCORS(w, r)
	if origin == "" {
		return
	}
	to := conf.HostContacts(host)
	if len(to) <= 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	msg := contactData{}
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("could not decode request body"))
		return
	}

	// Log first
	logEvent := db.MailLog{
		When:    time.Now(),
		Host:    host,
		Name:    msg.Name,
		Org:     msg.Org,
		Details: msg.Details,
		Msg:     msg.Msg,
		IP:      requestIP(r),
	}
	if err := db.DB.Create(&logEvent).Error; err != nil {
		log.Printf("Could not log contact data: %v", err)
	}
	sitedata := metrics.GetSiteData(host)
	sitedata.EventCount[metrics.EV_EMAIL]++

	// Then send email.
	from := "web-contact@mkmba.nz" // Must be mkmba.nz until SES is out of sandbox.
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	tmpl, err := templates.Get("contactform.tmpl")
	if err != nil {
		log.Printf("Could not load email template: %v", err)
	} else {
		data := map[string]any{
			"From":  from,
			"To":    to,
			"Event": logEvent,
		}
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, data)
		if err != nil {
			log.Printf("Could not render email template: %v", err)
		} else {
			var auth smtp.Auth
			if os.Getenv("SMTP_USER") != "" {
				auth = smtp.PlainAuth("", os.Getenv("SMTP_USER"), os.Getenv("SMTP_PASS"), smtpHost)
			}
			err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, buf.Bytes())
			if err != nil {
				log.Printf("Failed to send email for %s to %s: %s", host, to, err)
			}
		}
	}

	writeCORSHeaders(w, r)
	w.WriteHeader(http.StatusOK)
}

func CollectMetric(w http.ResponseWriter, r *http.Request) {
	origin, host := checkOriginCORS(w, r)
	if origin == "" {
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

	page := event.Page
	if page == "" {
		// Everything should sent us Page ideally, but if not
		// see if we can get it from the Referer header.
		page = r.Header.Get("Referer")
	}
	if page == "" {
		page = origin
	}
	referer := event.Referer
	if referer == page {
		referer = "" // Don't both storing referer if its the triggering page.
	}

	ip := requestIP(r)
	if conf.IsIgnoredIP(ip) {
		log.Printf("Ignoring %v on %s from ignored IP %s", event, page, ip)
	} else {
		// Trim page/referer from raw_event saved to save DB space
		// (they're explicit columns)
		event.Page = ""
		event.Referer = ""
		logEvent := db.EventLog{
			When:        time.Now(),
			Host:        host,
			Page:        page,
			Referer:     referer,
			UserAgentID: db.GetUserAgentID(r.Header.Get("User-Agent")),
			IP:          ip,
			RawEvent:    event,
		}
		if err := db.DB.Create(&logEvent).Error; err != nil {
			log.Printf("Could not log raw event: %v", err)
		}
		sitedata := metrics.GetSiteData(host)
		sitedata.EventCount[event.Event]++
	}

	writeCORSHeaders(w, r)
	w.WriteHeader(http.StatusOK)
}

func setupPublicHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/", CollectMetric)
	mux.HandleFunc("/contact", ContactForm)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("all good"))
	})

	mux.Handle("/js/", http.StripPrefix("/js/", serveWithCORS(js.FileServer())))
}

func setupTSHandlers(mux *http.ServeMux) {
	// register a prometheus metric exporter
	collector := prom.Collector{}
	prometheus.MustRegister(collector)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/dashboard", reporting.Home)
	mux.HandleFunc("/dashboard/{site}", reporting.Site)
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
	reporting.SetConfig(conf)

	err = tailscale.Init(fmt.Sprintf("metrics-%s", envName()), conf.StateDirectory, 30*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to tailscale: %v", err)
	}

	setupPublicHandlers(http.DefaultServeMux)
	tsmux := http.NewServeMux()
	setupTSHandlers(tsmux)

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

	// Try and also listen on TS (for /metrics)
	wg.Add(1)
	go func() {
		tailscale.Serve(tsmux)
		wg.Done()
	}()

	wg.Wait()
}

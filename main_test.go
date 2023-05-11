// Copyright © 2023 Matt Brown.
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mattb.nz/web/metrics/metrics"
)

func TestLoadJSONConfig(t *testing.T) {
	config, err := loadConfig("testdata/goodconfig.json")
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	if len(config) != 2 {
		t.Error("Expected 2 sites, got", len(config))
	}
	if config[0].Host != "test.com" {
		t.Error("Expected test.com, got", config[0].Host)
	}
	if config[0].AllowedReferers[0] != "test.com" {
		t.Error("Expected test.com, got", config[0].AllowedReferers[0])
	}
	if config[1].Host != "another.com" {
		t.Error("Expected another.com, got", config[1].Host)
	}
	if config[1].AllowedReferers[0] != "test2.com" {
		t.Error("Expected test2.com, got", config[1].AllowedReferers[0])
	}

	_, err = loadConfig("testdata/badconfig.json")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	_, err = loadConfig("testdata/doesnotexist.json")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	_, err = loadConfig("testdata/notjson.json")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func loadTestConfig(t *testing.T) {
	var err error
	config, err = loadConfig("testdata/goodconfig.json")
	if err != nil {
		panic(err)
	}

}

func Test_CollectMetric(t *testing.T) {
	loadTestConfig(t)

	tests := []struct {
		method  string
		path    string
		referer string
		body    string
		code    int
	}{
		{"GET", "/", "localhost", "", http.StatusNotFound},
		{"GET", "/", "test.com", "", http.StatusBadRequest},
		{"POST", "/", "test.com", "", http.StatusBadRequest},
		{"POST", "/", "test.com", `{"event":"pageview"}`, http.StatusOK},
		{"POST", "/", "test.com", `{"event":"pageview"}`, http.StatusOK},
		{"POST", "/", "test.com", `{"event":"click"}`, http.StatusOK},
		{"POST", "/", "test.com", `{"event":"activity"}`, http.StatusOK},
		{"POST", "/", "test.com", `{"event":"somethingelse"}`, http.StatusBadRequest},
	}

	setupHandlers()

	for i, test := range tests {
		req, err := http.NewRequest(test.method, test.path, strings.NewReader(test.body))
		req.Header.Set("Referer", test.referer)
		if err != nil {
			t.Errorf("Test %d: Error creating request: %v", i, err)
			continue
		}

		// Create a response recorder.
		rr := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(rr, req)

		// Check the status code.
		if status := rr.Code; status != test.code {
			t.Errorf("Test %d: handler returned wrong status code: got %v want %v. Body: %s", i, status, test.code, rr.Body.String())
		}
	}

	sites := metrics.Sites
	if sites["test.com"].EventCount["pageview"] != 2 {
		t.Error("Expected 2 pageviews, got", sites["test.com"].EventCount["pageview"])
	}
	if sites["test.com"].EventCount["click"] != 1 {
		t.Error("Expected 1 click, got", sites["test.com"].EventCount["click"])
	}
	if sites["test.com"].EventCount["activity"] != 1 {
		t.Error("Expected 1 activity, got", sites["test.com"].EventCount["activity"])
	}

	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Error("Error creating request:", err)
		return
	}
	rr := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Error("handler returned wrong status code: got", status)
	}
	if !strings.Contains(rr.Body.String(), "events_total{event=\"pageview\",site=\"test.com\"} 2") {
		t.Error("Expected 2 pageviews, got", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "events_total{event=\"click\",site=\"test.com\"} 1") {
		t.Error("Expected 1 click, got", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "events_total{event=\"activity\",site=\"test.com\"} 1") {
		t.Error("Expected 1 activity, got", rr.Body.String())
	}
}

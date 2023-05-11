// Copyright © 2023 Matt Brown.
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mattb.nz/web/metrics/config"
	"mattb.nz/web/metrics/db"
	"mattb.nz/web/metrics/metrics"
)

func Test_CollectMetric(t *testing.T) {
	tconf, err := config.LoadConfig("config/testdata/goodconfig.json")
	if err != nil {
		panic(err)
	}
	if err := db.Init(tconf); err != nil {
		panic(err)
	}
	conf = tconf

	tests := []struct {
		method  string
		path    string
		referer string
		body    string
		code    int
	}{
		{"GET", "/", "http://localhost/", "", http.StatusNotFound},
		{"GET", "/", "http://test.com/", "", http.StatusBadRequest},
		{"POST", "/", "http://test.com/", "", http.StatusBadRequest},
		{"POST", "/", "http://test.com/", `{"event":"pageview"}`, http.StatusOK},
		{"POST", "/", "http://test.com/", `{"event":"pageview"}`, http.StatusOK},
		{"POST", "/", "http://test.com/", `{"event":"click"}`, http.StatusOK},
		{"POST", "/", "http://test.com/", `{"event":"activity"}`, http.StatusOK},
		{"POST", "/", "http://test.com/", `{"event":"somethingelse"}`, http.StatusBadRequest},
	}

	mux := http.NewServeMux()
	setupHandlers(mux)

	for i, test := range tests {
		req, err := http.NewRequest(test.method, test.path, strings.NewReader(test.body))
		req.Header.Set("Referer", test.referer)
		if err != nil {
			t.Errorf("Test %d: Error creating request: %v", i, err)
			continue
		}

		// Create a response recorder.
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

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
	mux.ServeHTTP(rr, req)
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

	var c int64
	if err := db.DB.Model(&db.EventLog{}).Count(&c).Error; err != nil {
		t.Error("Error counting events:", err)
	}
	if c != 4 {
		t.Error("Expected 4 events, got", c)
	}
}

// Even without a DB, we should still be able to post events
func Test_CollectMetric_NoDB(t *testing.T) {
	tconf, err := config.LoadConfig("config/testdata/goodconfig.json")
	if err != nil {
		panic(err)
	}
	if err := db.Init(tconf); err != nil {
		panic(err)
	}
	conf = tconf

	mux := http.NewServeMux()
	setupHandlers(mux)

	req, err := http.NewRequest("POST", "/", strings.NewReader(`{"event":"click"}`))
	req.Header.Set("Referer", "test.com")
	if err != nil {
		t.Errorf("Error creating request: %v", err)
		return
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Check the status code.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v. Body: %s", status, http.StatusOK, rr.Body.String())
	}
}

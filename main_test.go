// Copyright © 2023 Matt Brown.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	smtpmock "github.com/mocktools/go-smtp-mock/v2"

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
		method string
		path   string
		origin string
		ua     string
		body   string
		code   int
	}{
		{"GET", "/", "http://localhost", "", "", http.StatusNotFound},
		{"GET", "/", "http://test.com", "", "", http.StatusBadRequest},
		{"POST", "/", "http://test.com", "", "", http.StatusBadRequest},
		{"POST", "/", "http://test.com", "browser1", `{"event":"pageview"}`, http.StatusOK},
		{"POST", "/", "http://test.com", "browser1", `{"event":"pageview"}`, http.StatusOK},
		{"POST", "/", "http://test.com", "browser2", `{"event":"click"}`, http.StatusOK},
		{"POST", "/", "http://test.com", "browser3", `{"event":"activity"}`, http.StatusOK},
		{"POST", "/", "http://test.com", "", `{"event":"somethingelse"}`, http.StatusBadRequest},
	}

	mux := http.NewServeMux()
	setupPublicHandlers(mux)

	for i, test := range tests {
		req, err := http.NewRequest(test.method, test.path, strings.NewReader(test.body))
		req.Header.Set("Origin", test.origin)
		req.Header.Set("User-Agent", test.ua)
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
	tsmux := http.NewServeMux()
	setupTSHandlers(tsmux)
	rr := httptest.NewRecorder()
	tsmux.ServeHTTP(rr, req)
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
	if err := db.DB.Model(&db.UserAgent{}).Count(&c).Error; err != nil {
		t.Error("Error counting user agents:", err)
	}
	if c != 3 {
		t.Error("Expected 3 user agents, got", c)
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
	setupPublicHandlers(mux)

	req, err := http.NewRequest("POST", "/", strings.NewReader(`{"event":"click"}`))
	req.Header.Set("Origin", "http://test.com")
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

// Test contact submission functionality
func Test_ContactForm(t *testing.T) {
	tconf, err := config.LoadConfig("config/testdata/goodconfig.json")
	if err != nil {
		panic(err)
	}
	if err := db.Init(tconf); err != nil {
		panic(err)
	}
	conf = tconf

	server := smtpmock.New(smtpmock.ConfigurationAttr{
		LogToStdout:       true,
		LogServerActivity: true,
	})
	if err := server.Start(); err != nil {
		panic(err)
	}
	old_host := os.Getenv("SMTP_HOST")
	old_port := os.Getenv("SMTP_PORT")
	old_user := os.Getenv("SMTP_USER")
	defer func() {
		os.Setenv("SMTP_HOST", old_host)
		os.Setenv("SMTP_PORT", old_port)
		os.Setenv("SMTP_PORT", old_user)
		if err := server.Stop(); err != nil {
			panic(err)
		}
	}()
	os.Setenv("SMTP_HOST", "127.0.0.1")
	os.Setenv("SMTP_PORT", fmt.Sprintf("%d", server.PortNumber()))
	os.Setenv("SMTP_USER", "")

	tests := []struct {
		method string
		path   string
		origin string
		body   string
		code   int
	}{

		{"GET", "/contact", "http://test.com", "", http.StatusBadRequest},  // Requires Post
		{"POST", "/contact", "http://localhost", "", http.StatusNotFound},  // Unknown origin
		{"POST", "/contact", "http://test.com", "", http.StatusBadRequest}, // Requires form data
		{"POST", "/contact", "http://test.com", `{"name":"me", "org":"yep", "details":"a@b.com", "msg":"hi"}`, http.StatusOK},
		{"POST", "/contact", "http://test2.com", `{"name":"me", "org":"yep", "details":"a@b.com", "msg":"hi"}`, http.StatusServiceUnavailable}, // not configured
	}

	mux := http.NewServeMux()
	setupPublicHandlers(mux)

	for i, test := range tests {
		req, err := http.NewRequest(test.method, test.path, strings.NewReader(test.body))
		req.Header.Set("Origin", test.origin)
		req.RemoteAddr = "127.0.0.1:45678"
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

	// Should only have 1 message generated
	for i, msg := range server.Messages() {
		if !msg.IsConsistent() {
			t.Errorf("Email %d did not send successfully!", i+1)
		}
		msgData := msg.MsgRequest()
		if i != 0 {
			t.Errorf("Expected 1 email to be generated, but found messaged #%d: %s", i+1, msgData)
		}
		expect := "Contact form submission from test.com"
		if !strings.Contains(msgData, expect) {
			t.Errorf("Message %d did not contain '%s': %s", i+1, expect, msgData)
		}
	}
	var msgs []db.MailLog
	if err := db.DB.Model(&db.MailLog{}).Find(&msgs).Error; err != nil {
		panic(err)
	}
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message in DB, got %d", len(msgs))
	}
	if msgs[0].Host != "test.com" {
		t.Errorf("Expected DB message Host to be test.com, got %s", msgs[0].Host)
	}
	if msgs[0].IP != "127.0.0.1" {
		t.Errorf("Expected DB message Host to be 127.0.0.1, got %s", msgs[0].IP)
	}
	if msgs[0].Name != "me" {
		t.Errorf("Expected DB message Name to be me, got %s", msgs[0].Name)
	}
	if msgs[0].Org != "yep" {
		t.Errorf("Expected DB message Org to be yep, got %s", msgs[0].Org)
	}
	if msgs[0].Details != "a@b.com" {
		t.Errorf("Expected DB message Details to be a@b.com, got %s", msgs[0].Details)
	}
	if msgs[0].Msg != "hi" {
		t.Errorf("Expected DB message Msg to be hi, got %s", msgs[0].Msg)
	}

}

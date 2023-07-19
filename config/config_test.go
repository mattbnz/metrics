package config

import "testing"

func TestLoadJSONConfig(t *testing.T) {
	config, err := LoadConfig("testdata/goodconfig.json")
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	sites := config.Sites
	if len(sites) != 2 {
		t.Error("Expected 2 sites, got", len(sites))
	}
	if sites[0].Host != "test.com" {
		t.Error("Expected test.com, got", sites[0].Host)
	}
	if sites[0].AllowedOrigins[0] != "http://test.com" {
		t.Error("Expected http://test.com, got", sites[0].AllowedOrigins[0])
	}
	if sites[1].Host != "another.com" {
		t.Error("Expected another.com, got", sites[1].Host)
	}
	if sites[1].AllowedOrigins[0] != "http://test2.com" {
		t.Error("Expected http://test2.com, got", sites[1].AllowedOrigins[0])
	}

	_, err = LoadConfig("testdata/badconfig.json")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	_, err = LoadConfig("testdata/doesnotexist.json")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	_, err = LoadConfig("testdata/notjson.json")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	_, err = LoadConfig("testdata/badcidr.json")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func Test_GetHostForReferer(t *testing.T) {
	conf, err := LoadConfig("testdata/goodconfig.json")
	if err != nil {
		t.Error("Expected no error, got", err)
	}

	host := conf.GetHostForOrigin("http://test.com")
	if host != "test.com" {
		t.Error("Expected test.com, got", host)
	}

	host = conf.GetHostForOrigin("http://test2.com")
	if host != "another.com" {
		t.Error("Expected another.com, got", host)
	}

	host = conf.GetHostForOrigin("http://test3.com")
	if host != "" {
		t.Error("Expected empty string, got", host)
	}
}

func Test_IsIgnoredIP(t *testing.T) {
	conf, err := LoadConfig("testdata/goodconfig.json")
	if err != nil {
		t.Error("Expected no error, got", err)
	}

	if conf.IsIgnoredIP("10.10.10.10") {
		t.Error("Expected 10.10.10.10 not to be ignored, but IsIgnoredIP returned true")
	}
	if !conf.IsIgnoredIP("10.10.11.10") {
		t.Error("Expected 10.10.11.10 to be ignored, but IsIgnoredIP returned false")
	}
	if !conf.IsIgnoredIP("10.10.11.250") {
		t.Error("Expected 10.10.11.250 to be ignored, but IsIgnoredIP returned false")
	}
	if !conf.IsIgnoredIP("192.168.1.2") {
		t.Error("Expected 192.168.1.2 to be ignored, but IsIgnoredIP returned false")
	}
}

func Test_HostContacts(t *testing.T) {
	conf, err := LoadConfig("testdata/goodconfig.json")
	if err != nil {
		t.Error("Expected no error, got", err)
	}

	contacts := conf.HostContacts("test.com")
	if len(contacts) != 1 || contacts[0] != "hi@test.com" {
		t.Error("Expected 1 contact (hi@test.com), got: ", contacts)
	}
	contacts = conf.HostContacts("another.com")
	if len(contacts) != 0 {
		t.Error("Expected no contacts, got: ", contacts)
	}
}

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
	if sites[0].AllowedReferers[0] != "test.com" {
		t.Error("Expected test.com, got", sites[0].AllowedReferers[0])
	}
	if sites[1].Host != "another.com" {
		t.Error("Expected another.com, got", sites[1].Host)
	}
	if sites[1].AllowedReferers[0] != "test2.com" {
		t.Error("Expected test2.com, got", sites[1].AllowedReferers[0])
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
}

func Test_GetHostForReferer(t *testing.T) {
	conf, err := LoadConfig("testdata/goodconfig.json")
	if err != nil {
		t.Error("Expected no error, got", err)
	}

	host := conf.GetHostForReferer("test.com")
	if host != "test.com" {
		t.Error("Expected test.com, got", host)
	}

	host = conf.GetHostForReferer("test2.com")
	if host != "another.com" {
		t.Error("Expected another.com, got", host)
	}

	host = conf.GetHostForReferer("test3.com")
	if host != "" {
		t.Error("Expected empty string, got", host)
	}
}

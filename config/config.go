package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

type MonitoredSite struct {
	Host           string
	AllowedOrigins []string
}

type Config struct {
	DatabaseUrl    string
	StateDirectory string

	Sites []MonitoredSite

	// List of networks to ignore requests from in CIDR notation
	IgnoreNets   []string
	_ignoredNets []*net.IPNet
}

// Load config from JSON file
func LoadConfig(filename string) (Config, error) {
	// Open the file.
	f, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	// Read the contents of the file into a byte array.
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return Config{}, err
	}

	// Unmarshal the JSON data into the provided interface.
	config := Config{}
	err = json.Unmarshal(b, &config)
	if err != nil {
		return Config{}, err
	}

	// Parse the ignored networks
	for _, cidrNet := range config.IgnoreNets {
		_, net, err := net.ParseCIDR(cidrNet)
		if err != nil {
			return Config{}, fmt.Errorf("could not parse ignored network %s: %v", cidrNet, err)
		}
		config._ignoredNets = append(config._ignoredNets, net)
	}

	return config, nil
}

func (c Config) GetHostForOrigin(origin string) string {
	for _, site := range c.Sites {
		for _, allowed := range site.AllowedOrigins {
			if strings.HasPrefix(origin, allowed) {
				return site.Host
			}
		}
	}
	return ""
}

func (c Config) IsIgnoredIP(ip string) bool {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}
	for _, net := range c._ignoredNets {
		if net.Contains(ipAddr) {
			return true
		}
	}
	return false
}

package config

import (
	"encoding/json"
	"io/ioutil"
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

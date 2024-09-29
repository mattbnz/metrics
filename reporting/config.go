package reporting

import "mattb.nz/web/metrics/config"

var conf config.Config

func SetConfig(c config.Config) {
	conf = c
}

func siteConfig(site string) config.MonitoredSite {
	for _, s := range conf.Sites {
		if s.Host == site {
			return s
		}
	}
	return config.MonitoredSite{}
}

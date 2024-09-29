package reporting

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"mattb.nz/web/metrics/config"
	"mattb.nz/web/metrics/db"
	"mattb.nz/web/metrics/metrics"
	"mattb.nz/web/metrics/templates"
)

func Site(w http.ResponseWriter, r *http.Request) {
	page, err := templates.Get("site.html")
	if err != nil {
		log.Printf("Could not load site page template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	site := r.PathValue("site")
	if site == "" {
		http.Error(w, "Bad Request", http.StatusNotFound)
		return
	}
	siteConfig := siteConfig(site)
	page.Execute(w, map[string]any{
		"Config":    conf,
		"Site":      site,
		"LiveData":  metrics.GetSiteData(site),
		"DayTotals": getDayTotals(siteConfig),
	})
}

type DayTotals map[int]SiteHistory

func getDayTotals(site config.MonitoredSite) (rv DayTotals) {
	rv = make(DayTotals)
	for _, days := range []int{1, 7, 30, 365} {
		rv[days] = siteHistory(site, days)
	}
	return rv
}

type SiteHistory map[string]any

func siteHistory(site config.MonitoredSite, days int) (rv SiteHistory) {
	rv = make(SiteHistory)
	v, err := db.Count(db.EventLog{}, "host = ? AND json_extract(raw_event, '$.Event') = ? AND `when` > ?", site.Host, metrics.EV_PAGEVIEW, time.Now().AddDate(0, 0, -days))
	if err != nil {
		rv["pageview"] = fmt.Sprintf("unavailable: %v", err)
	} else {
		rv["pageview"] = v
	}
	v, err = db.Count(db.EventLog{}, "host = ? AND json_extract(raw_event, '$.Event') = ? AND `when` > ?", site.Host, metrics.EV_ACTIVITY, time.Now().AddDate(0, 0, -days))
	if err != nil {
		rv["readtime"] = fmt.Sprintf("unavailable: %v", err)
	} else {
		rv["readtime"] = fmt.Sprintf("%d minutes", v)
	}
	rv["referers"] = siteReferers(site, days)
	return rv
}

type Referer struct {
	Referer string
	Count   int
}

func siteReferers(site config.MonitoredSite, days int) (rv []Referer) {
	rows, err := db.DB.Raw("SELECT referer, COUNT(*) AS count FROM event_logs WHERE host = ? AND json_extract(raw_event, '$.Event') = ? AND `when` > ? GROUP BY referer HAVING referer != '' ORDER BY count DESC LIMIT 10", site.Host, metrics.EV_PAGEVIEW, time.Now().AddDate(0, 0, -days)).Rows()
	if err != nil {
		log.Printf("Could not get site referers: %v", err)
		return rv
	}
	defer rows.Close()
	for rows.Next() {
		var referer string
		var count int
		if err := rows.Scan(&referer, &count); err != nil {
			log.Printf("Could not get site referers: %v", err)
			return rv
		}
		rv = append(rv, Referer{referer, count})
	}
	// Sort the results by count
	sort.Slice(rv, func(i, j int) bool {
		return rv[i].Count > rv[j].Count
	})
	return rv
}

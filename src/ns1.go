package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/alexcesaro/statsd.v2"
)

type NS1Config struct {
	APIKey string `yaml:"api_key"`
}

type NS1Provider struct {
	cfg *NS1Config
}

type ns1ZoneSummary struct {
	ID           string   `json:"id"`
	TTL          int      `json:"ttl"`
	NxTTL        int      `json:"nx_ttl"`
	Retry        int      `json:"retry"`
	Zone         string   `json:"zone"`
	Refresh      int      `json:"refresh"`
	Expiry       int      `json:"expiry"`
	DNSServers   []string `json:"dns_servers"`
	Networks     []int    `json:"networks"`
	NetworkPools []string `json:"network_pools"`
	Hostmaster   string   `json:"hostmaster"`
}

type ns1Zone struct {
	ns1ZoneSummary

	Primary struct {
		Enabled     bool          `json:"enabled"`
		Secondaries []interface{} `json:"secondaries"`
	} `json:"primary"`

	Secondary struct {
		// Status indicates whether or not the last poll to the master succeeded.
		// It initially starts as "pending" until after the first AXFR request.
		// After successfully pulling the zone it should switch to "ok,"
		// and if it ever has an issues polling the master it will indicate "warning."
		Status      string `json:"status"`
		LastXfr     int64  `json:"last_xfr"`
		PrimaryIP   string `json:"primary_ip"`
		PrimaryPort int    `json:"primary_port"`
		Enabled     bool   `json:"enabled"`
		Error       string `json:"error"`
		Expired     bool   `json:"expired"`
	} `json:"secondary"`

	Records []struct {
		ID           string   `json:"id"`
		Type         string   `json:"type"`
		Tier         int      `json:"tier"`
		TTL          int      `json:"ttl"`
		ShortAnswers []string `json:"short_answers"`
		Domain       string   `json:"domain"`
	} `json:"records"`
}

type ns1InstantQps struct {
	Qps float64 `json:"qps"`
}

func (p NS1Provider) CollectMetrics(rep *statsd.Client) (err error) {
	if p.cfg.APIKey == "" {
		err = errors.New("NS1 API key is not set")
		return
	}

	zones, err := p.getZones()
	if err != nil {
		err = errors.New("Cannot retrieve the zone list")
		return
	}

	for _, zone := range zones {
		log.Debug("NS1 provider is processing zone ", zone.Zone)
		r := rep.Clone(statsd.Tags("zone", zone.Zone, "provider", "ns1"))

		z, err := p.getZoneDetails(zone.Zone)
		if err != nil {
			log.Info("Error fetching zone ", zone.Zone, ": ", err)
		} else {
			p.reportZoneState(z, r)
		}

		qps, err := p.getInstantQps(zone.Zone)
		if err == nil {
			r.Gauge("zone.qps", qps)
		}
	}
	return
}

func (p NS1Provider) reportZoneState(z *ns1Zone, rep *statsd.Client) {
	rep.Gauge("zone.type.primary", BoolToInt(z.Primary.Enabled))
	rep.Gauge("zone.record_count", len(z.Records))

	if z.Secondary.Enabled {
		rep.Gauge("zone.type.secondary", 1)
		rep.Gauge("zone.secondary.is_ok", BoolToInt(z.Secondary.Status == "ok"))
		rep.Gauge("zone.secondary.sec_since_last_xfr", time.Now().Unix()-z.Secondary.LastXfr)
		rep.Gauge("zone.secondary.is_expired", BoolToInt(z.Secondary.Expired))
	} else {
		rep.Gauge("zone.type.secondary", 0)
	}
}

func (p NS1Provider) getZones() (zones []ns1ZoneSummary, err error) {
	req, err := http.NewRequest("GET", "https://api.nsone.net/v1/zones", nil)
	if err != nil {
		log.Debug("NS1 getZones NewRequest: ", err)
		return
	}
	req.Header.Set("X-NSONE-Key", url.QueryEscape(p.cfg.APIKey))
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Debug("NS1 getZones Do: ", err)
		return
	}
	if resp.StatusCode != 200 {
		log.Debug(resp.Status)
		return
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&zones)
	if err != nil {
		log.Debug("NS1 getZones response cannot be decoded: ", err)
		log.Debug(resp.Body)
		return
	}

	return
}

func (p NS1Provider) getZoneDetails(zoneName string) (zone *ns1Zone, err error) {
	safeZoneName := url.QueryEscape(zoneName)
	endpoint := fmt.Sprintf("https://api.nsone.net/v1/zones/%s", safeZoneName)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Debug("NS1 getZoneDetails NewRequest: ", err)
		return
	}
	req.Header.Set("X-NSONE-Key", url.QueryEscape(p.cfg.APIKey))
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Debug("NS1 getZones Do: ", err)
		return
	}
	if resp.StatusCode != 200 {
		log.Debug(resp.Status)
		return
	}
	defer resp.Body.Close()

	zone = new(ns1Zone)
	err = json.NewDecoder(resp.Body).Decode(zone)
	if err != nil {
		log.Debug("NS1 getZoneDetails response cannot be decoded: ", err)
		log.Debug(resp.Body)
		return
	}

	return
}

func (p NS1Provider) getInstantQps(zoneName string) (qps float64, err error) {
	safeZoneName := url.QueryEscape(zoneName)
	endpoint := fmt.Sprintf("https://api.nsone.net/v1/stats/qps/%s", safeZoneName)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Debug("NS1 getInstantQps NewRequest: ", err)
		return
	}
	req.Header.Set("X-NSONE-Key", url.QueryEscape(p.cfg.APIKey))
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Debug("NS1 getInstantQps Do: ", err)
		return
	}
	if resp.StatusCode != 200 {
		log.Debug(resp.Status)
		return
	}
	defer resp.Body.Close()

	qpsStruct := new(ns1InstantQps)
	err = json.NewDecoder(resp.Body).Decode(qpsStruct)
	if err != nil {
		log.Debug("NS1 getInstantQps response cannot be decoded: ", err)
		log.Debug(resp.Body)
		return
	}

	qps = qpsStruct.Qps
	return
}

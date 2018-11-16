package main

import (
	"encoding/csv" // for QPS report parsing
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/nesv/go-dynect/dynect"
	statsd "gopkg.in/alexcesaro/statsd.v2"
	"io"
	"sort"
	"strconv"
	"strings"
	//"net/http"
	"net/url"
	"time"
)

type DynConfig struct {
	Customer, Username, Password string
}

type DynProvider struct {
	cfg    *DynConfig
	client *dynect.Client
}

type DynQpsCsv struct {
	Csv string `json:"csv"`
}

type DynQpsResponse struct {
	Response dynect.ResponseBlock
	Data     DynQpsCsv `json:"data"`
}

func (p DynProvider) CollectMetrics(rep *statsd.Client) (err error) {
	if p.cfg.Customer == "" || p.cfg.Username == "" || p.cfg.Password == "" {
		err = errors.New("Dyn provider authentication is incomplete. Customer, username, and password fields are required.")
		return
	}

	p.client = dynect.NewClient(p.cfg.Customer)
	err = p.client.Login(p.cfg.Username, p.cfg.Password)
	if err != nil {
		return
	}

	zones, err := p.getZones()
	if err != nil {
		err = errors.New("Cannot retrieve the zone list")
		return
	}

	var qpsForZone map[string]float64
	qpsForZone, err = p.getQpsReport()
	if err != nil {
		log.Info("Error fetching QPS report: ", err)
	}

	for _, zone := range zones {
		// A zone returned by API actually looks like /REST/Zone/example.com/
		zone = strings.Replace(zone, "/REST/Zone/", "", 1)
		zone = strings.TrimRight(zone, "/")
		taggedRep := rep.Clone(statsd.Tags("zone", zone, "provider", "dyn"))

		log.Debug("Dyn provider is processing zone ", zone)
		var z *dynect.ZoneResponse
		z, err = p.getZoneDetails(zone)
		if err != nil {
			log.Info("Error fetching zone ", zone, ": ", err)
		}
		p.reportZoneState(&z.Data, taggedRep)

		var records *dynect.AllRecordsResponse
		records, err = p.getZoneRecords(zone)
		if err != nil {
			log.Info("Error fetching list of records for zone ", zone, ": ", err)
		}
		p.reportRecordsMetrics(records, zone, taggedRep)

		if qps, exists := qpsForZone[zone]; exists {
			taggedRep.Gauge("zone.qps", qps)
		} else {
			log.Debug("No Dyn QPS data for zone ", zone)
		}
	}
	return
}

func (p DynProvider) getZones() (zones []string, err error) {
	var response dynect.ZonesResponse
	err = p.client.Do("GET", "Zone", nil, &response)
	if err == nil {
		zones = response.Data
	}
	return
}

func (p DynProvider) getZoneDetails(zoneName string) (zone *dynect.ZoneResponse, err error) {
	zone = new(dynect.ZoneResponse)
	safeZoneName := url.QueryEscape(zoneName)
	endpoint := fmt.Sprintf("Zone/%s", safeZoneName)
	err = p.client.Do("GET", endpoint, nil, zone)
	return
}

func (p DynProvider) getZoneRecords(zoneName string) (records *dynect.AllRecordsResponse, err error) {
	records = new(dynect.AllRecordsResponse)
	safeZoneName := url.QueryEscape(zoneName)
	endpoint := fmt.Sprintf("AllRecord/%s", safeZoneName)
	err = p.client.Do("GET", endpoint, nil, records)
	return
}

func (p DynProvider) reportZoneState(z *dynect.ZoneDataBlock, rep *statsd.Client) {
	rep.Gauge("zone.type.primary", BoolToInt(z.ZoneType == "Primary"))
	rep.Gauge("zone.type.secondary", BoolToInt(z.ZoneType == "Secondary"))
	rep.Gauge("zone.serial", z.Serial)
}

func (p DynProvider) reportRecordsMetrics(records *dynect.AllRecordsResponse, zone string, rep *statsd.Client) {
	rep.Gauge("zone.record_count", len(records.Data))
}

func (p DynProvider) getQpsReport() (qpsForZone map[string]float64, err error) {
	qpsForZone = make(map[string]float64)

	// Fetch metrics from the API
	args := map[string]string{
		"start_ts":  strconv.FormatInt(time.Now().Unix()-15*60, 10),
		"end_ts":    strconv.FormatInt(time.Now().Unix(), 10),
		"breakdown": "zones",
	}
	q := new(DynQpsResponse)
	err = p.client.Do("POST", "QPSReport", args, q)
	if err != nil {
		return
	}

	raw, err := parseQpsCsv(q.Data.Csv)
	if err != nil {
		return qpsForZone, err
	}

	if len(raw) == 0 {
		err = errors.New("No QPS data received")
		return
	}

	// Return the most recent complete-period set of metrics (second last)
	qpsForZone = extractSecondLastQps(raw)
	return
}

func parseQpsCsv(csvInput string) (raw map[string]map[string]float64, err error) {
	r := csv.NewReader(strings.NewReader(csvInput))
	_, err = r.Read() // skip header: Timestamp Zone Queries
	raw = make(map[string]map[string]float64)

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return raw, err
		}
		entriesForTimestamp, exists := raw[rec[0]]
		if !exists {
			entriesForTimestamp = make(map[string]float64)
			raw[rec[0]] = entriesForTimestamp
		}
		qps, err := strconv.Atoi(rec[2])
		if err != nil {
			return raw, err
		}
		entriesForTimestamp[rec[1]] = float64(qps) / 300 // csv data reports count in 5-min interval
	}

	return
}

func extractSecondLastQps(raw map[string]map[string]float64) map[string]float64 {
	keys := make([]string, 0, len(raw))
	i := 0
	for key, _ := range raw {
		keys = append(keys, key)
		i++
	}
	sort.Strings(keys)

	return raw[keys[len(keys)-2]]
}

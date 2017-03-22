package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	statsd "gopkg.in/alexcesaro/statsd.v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

type Config struct {
	Providers        []string
	NS1              NS1Config
	Dyn              DynConfig
	CheckInterval    time.Duration
	CheckIntervalStr string `yaml:"check_interval"`
	StatsdAddress    string `yaml:"statsd_address"`
}

type DNSProvider interface {
	CollectMetrics(rep *statsd.Client) (err error)
}

func loadConfig(fileName *string) (cfg *Config) {
	data, err := ioutil.ReadFile(*fileName)
	if err != nil {
		log.Fatalln("Failed to read config file", err)
	}

	cfg = new(Config)
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalln("Failed to parse config file", err)
	}

	cfg.CheckInterval, err = time.ParseDuration(cfg.CheckIntervalStr)
	if err != nil {
		log.Fatalln("Cannot decode check_interval", err)
	}

	if cfg.StatsdAddress == "" {
		cfg.StatsdAddress = "localhost:8125"
	}

	return
}

func collectMetrics(cfg *Config, rep *statsd.Client) {
	if providerEnabled(cfg, "dyn") {
		dyn := DynProvider{&cfg.Dyn, nil}
		err := dyn.CollectMetrics(rep)
		if err != nil {
			log.Info("Dyn Provider metrics collection was unsuccessful: ", err)
		}
	}

	if providerEnabled(cfg, "ns1") {
		ns1 := NS1Provider{&cfg.NS1}
		err := ns1.CollectMetrics(rep)
		if err != nil {
			log.Info("NS1 Provider metrics collection was unsuccessful: ", err)
		}
	}
}

func providerEnabled(cfg *Config, provider string) bool {
	for _, p := range cfg.Providers {
		if p == provider {
			return true
		}
	}
	return false
}

func main() {
	configFileName := flag.String("config", "config.yml", "Path to the configuration file")
	once := flag.Bool("once", false, "Run one check immediately, then exit")
	debug := flag.Bool("debug", false, "Enable debugging output")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	cfg := loadConfig(configFileName)
	rep, err := CreateStatsdReporter(cfg, *once)
	if err != nil {
		log.Fatalln("Can't create a StatsD reporter:", err, " using address: ", cfg.StatsdAddress)
	}

	if *once {
		collectMetrics(cfg, rep)
	} else {
		ticker := time.NewTicker(cfg.CheckInterval)
		for _ = range ticker.C {
			collectMetrics(cfg, rep)
		}
	}
}

package main

import (
	"bufio"
	"bytes"
	"net"

	log "github.com/sirupsen/logrus"
	"gopkg.in/alexcesaro/statsd.v2"
)

func CreateStatsdReporter(cfg *Config, once bool) (rep *statsd.Client, err error) {
	var address string

	if once {
		address = CreateStatsdListener().String()
	} else {
		address = cfg.StatsdAddress
	}

	rep, err = statsd.New(
		statsd.Prefix("dnsmetrics"),
		statsd.Address(address),
		statsd.TagsFormat(cfg.StatsdTagFormat))
	return
}

func CreateStatsdListener() net.Addr {
	addr, _ := net.ResolveUDPAddr("udp", "localhost:0")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalln("Error creating StatsD listener: ", err)
	}
	go StatsdPrinterLoop(conn)

	return conn.LocalAddr()
}

func StatsdPrinterLoop(c *net.UDPConn) {
	buf := make([]byte, 2048)
	log.Debug("StatsD listener is ready on ", c.LocalAddr().String())

	for {
		n, _, err := c.ReadFromUDP(buf)
		if err != nil {
			log.Info("Error reading from StatsD UDP channel: ", err)
			return
		}

		if n > 0 {
			reader := bytes.NewReader(buf[:n])
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				log.Debug("StatsD message: ", scanner.Text())
			}
		}
	}
}

*DNSmetrics* connects to your accounts at multiple managed DNS providers 
using their APIs and emits standardized metrics in statsd format for easy 
ingestion into your monitoring solution of choice.

While any DNS provider has a control panel, their UI can be incomplete
and they might not have alerting on metrics that are important to you. 
Furthermore, if you use several providers for redundancy, there is no way
to have one dashboard that shows unified health of your DNS architecture.

# Supported DNS providers

- [Dyn](http://dyn.com/)
- [NS1](https://ns1.com/)

It is easy to add new providers. Contributions are most welcome.

# Installation

With Go 1.7 installed, do `make install` and this will result in a binary.

# Configuration

Take a look at `config.yml`. It should be self explanatory.

# Running DNSmetrics

For testing, using `--once` will output the collected metrics to stdout
instead of sending them to statsd. Without `--once`, DNSmetrics will run
as a service, collecting and sending metrics every `check_interval`.

# List of Metrics

All metrics are tagged with `zone` and `provider` tags. Zone is the DNS
zone for the metric and provider is the DNS provider. 

All metrics have the prefix `dnsmetrics`.

Not all providers support all metrics described below.

- *`zone.type.primary`, `zone.type.secondary`* - 0 or 1.

- *`zone.qps`* - current (or most recent) rate of queries per second,
  of all types, for the zone.

- *`zone.record_count`* - number of records of all types in the zone.

- *`zone.serial`* - the serial number of the zone.

- *`zone.secondary.is_ok`* - 0 or 1. 1 when provider considers the zone
  to be healthy.

- *`zone.secondary.is_expired`* - 0 or 1. 1 when the zone is expired and
  presumably no longer being served by the provider.

- *`zone.secondary.sec_since_last_xfr`* - time since last zone transfer.

# Contributing

Fork it, create a new feature branch, make your changes, open a pull request.
Tests would be much appreciated.

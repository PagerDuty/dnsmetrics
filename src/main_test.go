package main

import (
	"testing"
)

func TestProviderEnabled(t *testing.T) {
	cfg := Config{}
	providers := [2]string{"foo", "bar"}
	cfg.Providers = providers[:]

	if providerEnabled(&cfg, "eek") {
		t.Error("eek provider should not be enabled")
	}
	if !providerEnabled(&cfg, "bar") {
		t.Error("bar provider should be enabled")
	}
}

package ui

import (
	"testing"
	"time"
)

func TestApplyNetRates(t *testing.T) {
	t0 := time.Now()
	key := guestSampleKey{Cluster: "default", Node: "pve1", VMID: 100}

	// First sample: no previous data, so rate is unknown/zero rather than a
	// spike from diffing against zero.
	first := []guest{{Cluster: "default", Node: "pve1", VMID: 100, NetIn: 1000, NetOut: 500}}
	samples := applyNetRates(first, nil, t0)

	if first[0].NetInRate != 0 || first[0].NetOutRate != 0 {
		t.Fatalf("expected zero rate on first sample, got in=%v out=%v", first[0].NetInRate, first[0].NetOutRate)
	}
	if got := samples[key]; got.NetIn != 1000 || got.NetOut != 500 {
		t.Fatalf("expected sample to record cumulative counters, got %+v", got)
	}

	// Second sample, 2 seconds later: 2000 bytes in over 2s = 1000 B/s.
	t1 := t0.Add(2 * time.Second)
	second := []guest{{Cluster: "default", Node: "pve1", VMID: 100, NetIn: 3000, NetOut: 1500}}
	samples = applyNetRates(second, samples, t1)

	if second[0].NetInRate != 1000 {
		t.Fatalf("expected NetInRate 1000 B/s, got %v", second[0].NetInRate)
	}
	if second[0].NetOutRate != 500 {
		t.Fatalf("expected NetOutRate 500 B/s, got %v", second[0].NetOutRate)
	}

	// Counter went backwards (guest restarted) — must not report a bogus
	// negative-turned-huge rate.
	t2 := t1.Add(2 * time.Second)
	restarted := []guest{{Cluster: "default", Node: "pve1", VMID: 100, NetIn: 50, NetOut: 20}}
	applyNetRates(restarted, samples, t2)

	if restarted[0].NetInRate != 0 || restarted[0].NetOutRate != 0 {
		t.Fatalf("expected zero rate after counter reset, got in=%v out=%v", restarted[0].NetInRate, restarted[0].NetOutRate)
	}
}

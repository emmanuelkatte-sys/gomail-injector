package core

import (
	"math/rand"
	"testing"
	"time"

	"__MODULE_PLACEHOLDER__/config"
)

func TestSendDelay_zero(t *testing.T) {
	if d := sendDelay(0, 0, rand.New(rand.NewSource(1))); d != 0 {
		t.Fatalf("expected 0, got %v", d)
	}
}

func TestSendDelay_fixed(t *testing.T) {
	if d := sendDelay(500, 0, rand.New(rand.NewSource(1))); d != 500*time.Millisecond {
		t.Fatalf("expected 500ms, got %v", d)
	}
}

func TestSendDelay_range(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 50; i++ {
		d := sendDelay(100, 200, rng)
		ms := d.Milliseconds()
		if ms < 100 || ms > 200 {
			t.Fatalf("delay out of range: %dms", ms)
		}
	}
}

func TestPerformanceNormalizedIntervals_legacyYAML(t *testing.T) {
	p := (&config.PerformanceConfig{
		IntervalMinMs: 300,
		IntervalMaxMs: 900,
	}).NormalizedIntervals()
	if p.MinInterval != 300 || p.MaxInterval != 900 {
		t.Fatalf("legacy keys not applied: %+v", p)
	}
}

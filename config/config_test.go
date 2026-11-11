package config

import (
	"testing"
)

func TestAppStrategy(t *testing.T) {
	app := App{}
	app.Init()

	if app.Strategy == nil {
		t.Fatal("Strategy should not be nil")
	}
	if app.Strategy["allow-hide-cm"] != "Y" {
		t.Fatalf("expected allow-hide-cm to be Y, got %v", app.Strategy["allow-hide-cm"])
	}

	ts := app.GetStrategyTimestamp()
	if ts <= 0 {
		t.Fatalf("expected positive timestamp, got %d", ts)
	}

	// Disable hide-cm
	f := false
	app2 := App{HideCm: &f}
	app2.Init()
	if _, ok := app2.Strategy["allow-hide-cm"]; ok {
		t.Fatal("expected allow-hide-cm to be absent when HideCm is false")
	}
	if app2.GetStrategyTimestamp() != 0 {
		t.Fatalf("expected 0 timestamp for empty strategy, got %d", app2.GetStrategyTimestamp())
	}
}

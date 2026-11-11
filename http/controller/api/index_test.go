package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
)

func TestHeartbeat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	global.Config.App.Init()

	r := gin.New()
	idx := &Index{}
	r.POST("/api/heartbeat", idx.Heartbeat)

	// Case 1: First heartbeat with modified_at = 0
	body1 := []byte(`{"id":"test-peer","uuid":"test-uuid","ver":100,"modified_at":0}`)
	req1 := httptest.NewRequest("POST", "/api/heartbeat", bytes.NewBuffer(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	var resp1 map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &resp1)
	if err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	ts, ok := resp1["modified_at"].(float64)
	if !ok || ts <= 0 {
		t.Fatalf("expected positive modified_at, got %v", resp1["modified_at"])
	}

	strat, ok := resp1["strategy"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected strategy in response, got %v", resp1)
	}
	opts, ok := strat["config_options"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected config_options in strategy, got %v", strat)
	}
	if opts["allow-hide-cm"] != "Y" {
		t.Fatalf("expected allow-hide-cm == Y, got %v", opts["allow-hide-cm"])
	}

	// Case 2: Subsequent heartbeat with matching modified_at
	body2, _ := json.Marshal(map[string]interface{}{
		"id":          "test-peer",
		"uuid":        "test-uuid",
		"ver":         100,
		"modified_at": int64(ts),
	})
	req2 := httptest.NewRequest("POST", "/api/heartbeat", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp2 map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if _, hasStrat := resp2["strategy"]; hasStrat {
		t.Fatalf("expected no strategy when modified_at matches, got %v", resp2)
	}
	if resp2["modified_at"] != ts {
		t.Fatalf("expected modified_at == %v, got %v", ts, resp2["modified_at"])
	}

	// Case 3: Disable hide-cm
	f := false
	global.Config.App.HideCm = &f
	global.Config.App.Init()

	body3 := []byte(`{"id":"test-peer","uuid":"test-uuid","ver":100,"modified_at":0}`)
	req3 := httptest.NewRequest("POST", "/api/heartbeat", bytes.NewBuffer(body3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	var resp3 map[string]interface{}
	_ = json.Unmarshal(w3.Body.Bytes(), &resp3)
	if resp3["modified_at"].(float64) != 0 {
		t.Fatalf("expected 0 modified_at when strategy empty, got %v", resp3["modified_at"])
	}
	if _, hasStrat := resp3["strategy"]; hasStrat {
		t.Fatalf("expected no strategy when empty, got %v", resp3)
	}
}

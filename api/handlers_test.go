package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fleet-metrics/models"
	"fleet-metrics/store"
)

func setupTestServer() *Server {
	s := store.NewStore()
	s.RegisterDevice("dev-1")
	return NewServer(s)
}

func TestHandleHeartbeat_Success(t *testing.T) {
	srv := setupTestServer()

	reqBody := models.HeartbeatRequest{SentAt: time.Now()}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/dev-1/heartbeat", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/devices/{device_id}/heartbeat", srv.HandleHeartbeat)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %v", w.Code)
	}

	_, _, err := srv.Store.GetStats("dev-1")
	if err != nil {
		t.Fatalf("expected heartbeat to be recorded, but it was not")
	}
}

func TestHandleStats_GetSuccess(t *testing.T) {
	srv := setupTestServer()

	deviceID := "dev-1"
	srv.Store.RecordHeartbeat(deviceID, time.Now())
	srv.Store.RecordUpload(deviceID, time.Now(), int64(5*time.Second))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/stats", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/devices/{device_id}/stats", srv.HandleStats)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %v. body: %s", w.Code, w.Body.String())
	}

	var resp models.GetDeviceStatsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Uptime != 100.0 {
		t.Errorf("expected 100 uptime, got %v", resp.Uptime)
	}
	if resp.AvgUploadTime != "5s" {
		t.Errorf("expected 5s avg upload time, got %v", resp.AvgUploadTime)
	}
}

func TestHandleStats_PostSuccess(t *testing.T) {
	srv := setupTestServer()

	reqBody := models.UploadStatsRequest{SentAt: time.Now(), UploadTime: int64(1 * time.Second)}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/dev-1/stats", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/devices/{device_id}/stats", srv.HandleStats)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %v", w.Code)
	}
}

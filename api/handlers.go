package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"fleet-metrics/models"
	"fleet-metrics/store"
)

// Server holds the store and registers HTTP routes
type Server struct {
	Store *store.ServerStore
}

// NewServer creates a new API Server instance
func NewServer(s *store.ServerStore) *Server {
	return &Server{
		Store: s,
	}
}

// writeError Helper for returning 404 or 500 errors matching openapi.json
func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"msg": msg})
}

// HandleHeartbeat handles POST /api/v1/devices/{device_id}/heartbeat
func (s *Server) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	var req models.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusInternalServerError, "invalid request body")
		return
	}

	s.Store.RecordHeartbeat(deviceID, req.SentAt)

	w.WriteHeader(http.StatusNoContent)
}

// HandleStats handles both POST and GET for /api/v1/devices/{device_id}/stats
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handlePostStats(deviceID, w, r)
	case http.MethodGet:
		s.handleGetStats(deviceID, w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handlePostStats POST /api/v1/devices/{device_id}/stats
func (s *Server) handlePostStats(deviceID string, w http.ResponseWriter, r *http.Request) {
	var req models.UploadStatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusInternalServerError, "invalid request body")
		return
	}

	s.Store.RecordUpload(deviceID, req.SentAt, req.UploadTime)
	w.WriteHeader(http.StatusNoContent)
}

// handleGetStats GET /api/v1/devices/{device_id}/stats
func (s *Server) handleGetStats(deviceID string, w http.ResponseWriter, r *http.Request) {
	uptime, avgUpload, err := s.Store.GetStats(deviceID)
	if err != nil {
		if fmt.Sprint(err) == "device not found" || fmt.Sprint(err) == "no heartbeats found for device" {
			writeError(w, http.StatusNotFound, "Device not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Format response
	// The API spec asks for AvgUploadTime as a string e.g. "5m10s"
	resp := models.GetDeviceStatsResponse{
		AvgUploadTime: avgUpload.String(),
		Uptime:        uptime,
	}

	// Calculate and write results
	resultText := fmt.Sprintf("[%s] uptime %f%% | avgUploadTime %s\n", deviceID, resp.Uptime, resp.AvgUploadTime)
	
	// Write to console
	fmt.Print(resultText)

	// Append to results.txt
	f, err := os.OpenFile("results.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error writing to results.txt: %v\n", err)
	} else {
		defer f.Close()
		if _, err := f.WriteString(resultText); err != nil {
			fmt.Printf("Error appending to results.txt: %v\n", err)
		}
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf("Error encoding stats response: %v\n", err)
	}
}

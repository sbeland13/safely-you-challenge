package models

import "time"

// HeartbeatRequest represents the body of a POST /heartbeat request
type HeartbeatRequest struct {
	SentAt time.Time `json:"sent_at"`
}

// UploadStatsRequest represents the body of a POST /stats request
type UploadStatsRequest struct {
	SentAt     time.Time `json:"sent_at"`
	UploadTime int64     `json:"upload_time"`
}

// GetDeviceStatsResponse represents the response of a GET /stats request
type GetDeviceStatsResponse struct {
	AvgUploadTime string  `json:"avg_upload_time"`
	Uptime        float64 `json:"uptime"`
}

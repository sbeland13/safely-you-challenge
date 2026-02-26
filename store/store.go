package store

import (
	"fmt"
	"sync"
	"time"
)

// DeviceData holds the metrics for a single device
type DeviceData struct {
	mu            sync.RWMutex
	Heartbeats    map[time.Time]bool
	UploadTimes   []int64
	FirstHeartbeat time.Time
	LastHeartbeat  time.Time
}

// ServerStore is the main thread-safe data store
type ServerStore struct {
	DeviceMap sync.Map // map[string]*DeviceData
}

// NewStore creates a new in-memory store
func NewStore() *ServerStore {
	return &ServerStore{}
}

// getOrCreateDevice retrieves or creates the data store for a given device ID
func (s *ServerStore) getOrCreateDevice(deviceID string) *DeviceData {
	data, _ := s.DeviceMap.LoadOrStore(deviceID, &DeviceData{})
	return data.(*DeviceData)
}

// RecordHeartbeat adds a heartbeat to a device's record
func (s *ServerStore) RecordHeartbeat(deviceID string, sentAt time.Time) {
	device := s.getOrCreateDevice(deviceID)
	device.mu.Lock()
	defer device.mu.Unlock()

	if device.Heartbeats == nil {
		device.Heartbeats = make(map[time.Time]bool)
	}
	device.Heartbeats[sentAt] = true

	if device.FirstHeartbeat.IsZero() || sentAt.Before(device.FirstHeartbeat) {
		device.FirstHeartbeat = sentAt
	}
	if device.LastHeartbeat.IsZero() || sentAt.After(device.LastHeartbeat) {
		device.LastHeartbeat = sentAt
	}
}

// RecordUpload adds an upload duration to a device's record
func (s *ServerStore) RecordUpload(deviceID string, sentAt time.Time, uploadTime int64) {
	device := s.getOrCreateDevice(deviceID)
	device.mu.Lock()
	defer device.mu.Unlock()

	device.UploadTimes = append(device.UploadTimes, uploadTime)
}

// GetStats calculates and returns the uptime and average upload time
// for a given device. Returns an error if the device has no records.
func (s *ServerStore) GetStats(deviceID string) (float64, time.Duration, error) {
	value, ok := s.DeviceMap.Load(deviceID)
	if !ok {
		return 0, 0, fmt.Errorf("device not found")
	}

	device := value.(*DeviceData)
	device.mu.RLock()
	defer device.mu.RUnlock()

	if len(device.Heartbeats) == 0 {
		return 0, 0, fmt.Errorf("no heartbeats found for device")
	}

	// 1. Calculate Uptime
	uptime := 100.0 // Default to 100% if only one heartbeat
	minutesDiff := device.LastHeartbeat.Sub(device.FirstHeartbeat).Minutes()

	if minutesDiff > 0 {
		// Calculate the number of full minutes
		numMinutes := float64(int(minutesDiff)) 
		
		// If exactly on a minute boundary or less than a minute, avoid zero division
		if numMinutes > 0 {
			// Each device sends a heartbeat every minute. We expected numMinutes + 1 heartbeats.
			// But the formula given is: uptime = (sumHeartbeats / numMinutesBetweenFirstAndLastHeartbeat) * 100
			uptime = (float64(len(device.Heartbeats)) / numMinutes) * 100
			// Cap at 100% just in case of duplicate heartbeats in the same minute
			if uptime > 100 {
				uptime = 100
			}
		}
	}

	// 2. Calculate Average Upload Time
	var avgUpload time.Duration
	if len(device.UploadTimes) > 0 {
		var sum int64
		for _, t := range device.UploadTimes {
			sum += t
		}
		// Calculate average in nanoseconds, then convert to time.Duration
		avgNanoseconds := sum / int64(len(device.UploadTimes))
		avgUpload = time.Duration(avgNanoseconds)
	}

	return uptime, avgUpload, nil
}

package store

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestStore_Heartbeats(t *testing.T) {
	s := NewStore()
	deviceID := "test-device"
	s.RegisterDevice(deviceID)
	now := time.Now()

	s.RecordHeartbeat(deviceID, now.Add(-2*time.Minute))
	s.RecordHeartbeat(deviceID, now.Add(-1*time.Minute))
	s.RecordHeartbeat(deviceID, now)

	uptime, _, err := s.GetStats(deviceID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 3 heartbeats over 2 minutes -> 100%
	if uptime != 100.0 {
		t.Errorf("expected 100.0 uptime, got %v", uptime)
	}
}

func TestStore_Heartbeats_Missing(t *testing.T) {
	s := NewStore()
	deviceID := "test-device"
	s.RegisterDevice(deviceID)
	now := time.Now()

	s.RecordHeartbeat(deviceID, now.Add(-5*time.Minute))
	s.RecordHeartbeat(deviceID, now.Add(-4*time.Minute))
	s.RecordHeartbeat(deviceID, now.Add(-3*time.Minute))
	// Misses 2 minutes
	s.RecordHeartbeat(deviceID, now)

	uptime, _, err := s.GetStats(deviceID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 4 heartbeats over 5 minutes: (4/5) * 100 = 80%
	if uptime != 80.0 {
		t.Errorf("expected 80.0 uptime, got %v", uptime)
	}
}

func TestStore_Heartbeats_OnePing(t *testing.T) {
	s := NewStore()
	deviceID := "test-device"
	s.RegisterDevice(deviceID)
	now := time.Now()

	s.RecordHeartbeat(deviceID, now)

	uptime, _, err := s.GetStats(deviceID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 1 heartbeat over 0 minutes -> fallback to 100%
	if uptime != 100.0 {
		t.Errorf("expected 100.0 uptime for single ping, got %v", uptime)
	}
}

func TestStore_UploadStats(t *testing.T) {
	s := NewStore()
	deviceID := "test-device"
	s.RegisterDevice(deviceID)
	now := time.Now()

	s.RecordHeartbeat(deviceID, now) // Needed otherwise GetStats fails

	s.RecordUpload(deviceID, now, int64(time.Second*5))
	s.RecordUpload(deviceID, now.Add(time.Second), int64(time.Second*10))

	_, avgUpload, err := s.GetStats(deviceID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedAvg := time.Second * 7 + time.Millisecond*500
	if avgUpload != expectedAvg {
		t.Errorf("expected avg upload time %v, got %v", expectedAvg, avgUpload)
	}
}

func TestStore_Concurrency(t *testing.T) {
	s := NewStore()
	deviceID := "test-device"
	s.RegisterDevice(deviceID)
	now := time.Now()

	var wg sync.WaitGroup
	workers := 100

	wg.Add(workers * 2)
	for i := 0; i < workers; i++ {
		go func(offset int) {
			defer wg.Done()
			s.RecordHeartbeat(deviceID, now.Add(time.Duration(offset)*time.Minute))
		}(i)
		go func(offset int) {
			defer wg.Done()
			s.RecordUpload(deviceID, now.Add(time.Duration(offset)*time.Minute), int64(offset*1000))
		}(i)
	}

	wg.Wait()

	// Should not crash or race
	_, _, err := s.GetStats(deviceID)
	if err != nil {
		t.Fatalf("unexpected error getting stats: %v", err)
	}
	fmt.Println("Concurrency test passed")
}

func BenchmarkGetStats(b *testing.B) {
	s := NewStore()
	deviceID := "test-device"
	s.RegisterDevice(deviceID)
	now := time.Now()

	s.RecordHeartbeat(deviceID, now.Add(-5*time.Minute))
	s.RecordHeartbeat(deviceID, now)
	s.RecordUpload(deviceID, now, int64(time.Second*5))

	b.ResetTimer() // Reset the timer so setup time isn't counted

	for b.Loop() {
		s.GetStats(deviceID)
	}
}

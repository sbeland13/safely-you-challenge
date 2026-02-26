package main

import (
	"fmt"
	"log"
	"net/http"

	"fleet-metrics/api"
	"fleet-metrics/store"
)

func main() {
	fmt.Println("Starting Fleet Metrics Server on port 6733...")
	
	serverStore := store.NewStore()
	if err := serverStore.LoadDevices("devices.csv"); err != nil {
		log.Fatalf("Warning: failed to load devices.csv: %v", err)
	}

	apiServer := api.NewServer(serverStore)

	// Route multiplexing
	mux := http.NewServeMux()
	
	// standard library handles wildcard routing nicely since 1.22
	mux.HandleFunc("POST /api/v1/devices/{device_id}/heartbeat", apiServer.HandleHeartbeat)
	mux.HandleFunc("POST /api/v1/devices/{device_id}/stats", apiServer.HandleStats)
	mux.HandleFunc("GET /api/v1/devices/{device_id}/stats", apiServer.HandleStats)

	if err := http.ListenAndServe("127.0.0.1:6733", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

# Fleet Management Metrics Server

This is a simple, yet highly performant implementation of the Fleet Management Metrics Server in Go.
It provides a set of endpoints designed to ingest device heartbeats and video upload times securely, maintaining and aggregating statistics thread-safe using standard Go structs and mutexes. All the endpoints match the provided `openapi.json` contract precisely.

## Requirements

You must have Go installed. The project was built and tested using **Go 1.26+**, which is required to utilize the newly enhanced standard library router `http.ServeMux` without requiring external routing dependencies like Gorilla Mux.

## Installation

1. Ensure Go 1.24+ is installed on your system.
2. Clone or navigate to the repository folder:
   ```bash
   cd safely-you-challenge
   ```
3. Initialize the modules and pull/tidy dependencies (though there are no external dependencies):
   ```bash
   go mod tidy
   ```

## Running the Server

To start the metrics server on port `6733`:
```bash
go run main.go
```
The server will bind to `127.0.0.1:6733` and accept incoming API requests.

## Running Tests
To run the unit test suite targeting stats calculation logic and handler parsing:
```bash
go test -v ./...
```
This tests for standard conditions and edge cases like 100% uptime with simply one single successful heartbeat.

## Running the Challenge Simulator

To execute the provided simulator against the running server and ensure correctness, open a new terminal window:
```bash
# First ensure the binary is executable (macOS arm64only)
chmod +x ./device-simulator-mac-arm64
# Execute with the -port flag
./device-simulator-mac-arm64 -port 6733
```

You will see the simulator verify that all responses precisely align with what is requested. 
The test generates a new file named `results.txt` displaying the metrics output in addition to printing nicely via the console.

## Approach & Design

1. **Routing:** Instead of utilizing a package like `gorilla/mux` or `chi`, this implementation explicitly takes advantage of `net/http`'s native multiplexer to retain the minimal possible footprint with 0 dependencies. Path extraction is cleanly abstracted inside handlers. Should this simple service become part of a large monolithic REST API with many more endpoints, I would likely opt for a router like `chi` for its middleware support and cleaner API.
2. **Concurrency Storage:** Storing and updating analytics stats implies extensive synchronized read/writes. `sync.Map` in conjunction with a specialized `sync.RWMutex` structure at the individual `DeviceData` level is utilized for absolute thread-safety, keeping reads fast and limiting mutex blocking purely to the device modifying its own state at the atomic level rather than completely locking down a global store map on every single request.
3. **Data Handling:** Timestamps are accurately stored explicitly as `time.Time` allowing natural extraction of differences `LastHeartbeat.Sub(FirstHeartbeat).Minutes()` avoiding arbitrary math.
4. **Idempotency:** The memory store is designed to be fully idempotent to gracefully handle duplicate requests and simulation re-runs. Instead of appending all incoming heartbeat data to an array—which would falsely inflate total counts for identical payload bursts—the heartbeats are stored in a boolean map keyed by `time.Time` (`map[time.Time]bool`). This guarantees that receiving multiple identical heartbeats for the exact same second from the same device mathematically evaluates correctly without skewing the uptime percentage. Conversely, `UploadTimes` are stored as a slice (`[]int64`); this preserves parallel uploads sharing identical timestamps, which is safe because computing the mathematical average over duplicate sets yields the exact same average naturally.

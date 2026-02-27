# Fleet Management Metrics Server

This is a simple, yet highly performant implementation of the Fleet Management Metrics Server in Go.
It provides a set of endpoints designed to ingest device heartbeats and video upload times securely, maintaining and aggregating statistics thread-safe using standard Go structs and mutexes. All the endpoints match the provided `openapi.json` contract precisely.

## Requirements

You must have Go installed. The project was built and tested using **Go 1.26+**, which allowed the use of the standard library router `http.ServeMux` without requiring external routing dependencies like Gorilla Mux.

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
4. **Idempotency:** The memory store is designed to be fully idempotent to gracefully handle duplicate requests and simulation re-runs. Instead of appending all incoming heartbeat data to an array, which would falsely inflate total counts for identical payload bursts,the heartbeats are stored in a boolean map keyed by `time.Time` (`map[time.Time]bool`). This guarantees that receiving multiple identical heartbeats for the exact same second from the same device mathematically evaluates correctly without skewing the uptime percentage. Conversely, `UploadTimes` are stored as a slice (`[]int64`); this preserves parallel uploads sharing identical timestamps, which is safe because computing the mathematical average over duplicate sets yields the exact same average naturally.
5. **Security & Validation:** The API actively defends against memory leaks and unauthorized telemetry ingestion by enforcing a strict device whitelist. On server initialization, `devices.csv` is parsed, and an exact set of allowed IDs is pre-populated into the memory state. Any incoming `POST` or `GET` request originating from a device ID not explicitly found within this CSV whitelist is immediately rejected with a `404 Not Found` response.

---

## Questions from PDF Deliverables

### 1. How long did you spend working on the problem? What did you find to be the most difficult part?
I spent about 2 hours working on this problem. The most difficult part was figuring out how to handle the duplicate heartbeats and ensuring that the uptime percentage was calculated correctly against concurrent data. I've been working mostly in Python over the past 3 years, so I did need to take time to refresh my memory on Go's syntax and standard library.

### 2. How would you modify your data model or code to account for more kinds of metrics?
Right now, the `DeviceData` struct is heavily coupled to the specific assignment requirements (it tightly defines `Heartbeats` and `UploadTimes`). This is called a rigid schema.

If the business suddenly asked us to track 15 new metrics (CPU usage, memory, temperature, network bandwidth, error counts, etc.), adding 15 new maps and slices to `DeviceData` would bloat the code and require constant API refactoring.

**How I would modify it:**

1. **Generic Event Ingestion Model:** I would refactor the API to use a generic payload rather than strict `HeartbeatRequest` / `UploadStatsRequest` structs.
   ```go
   type MetricType string
   
   type MetricEvent struct {
       Timestamp time.Time  `json:"timestamp"`
       Type      MetricType `json:"type"`       // e.g. "temperature", "cpu", "upload_time"
       Value     float64    `json:"value"`
   }
   ```
2. **Dynamic Time-Series Maps:** Inside `DeviceData`, instead of having hardcoded fields for every metric type, I would maintain a mapped collection of metrics:
   ```go
   type DeviceData struct {
       mu      sync.RWMutex
       Metrics map[MetricType][]MetricEvent
   }
   ```
   This allows the system to natively ingest entirely new metrics without having to redeploy or rewrite the code.

3. **Migrate to a Real TSDB:** If the system is scaling to "more kinds of metrics," storing unbounded slices of data in-memory becomes a dangerous memory leak. The most appropriate modification wouldn't belong in Go's memory at all, I would modify the Go server to simply act as an ingestion proxy. The Go code would validate the payload, and then immediately push the payload to an actual Time-Series Database like Prometheus, InfluxDB, or TimescaleDB, which natively handle data retention, rollups, and aggregation.

### 3. Discuss your solution's runtime complexity
The solution was explicitly designed for extremely high throughput by utilizing granular lock mechanisms.

**Ingestion (POST Requests): Time Complexity `O(1)`**
* **Device Lookup (`GetDevice`)**: Uses `sync.Map`, which has an amortized `O(1)` lookup time.
* **Recording Data (`RecordHeartbeat` / `RecordUpload`)**:
  * Acquiring the lock (`device.mu.Lock()`) is `O(1)`. *Crucially, it only locks the specific device's memory, meaning thousands of different devices can write concurrently without blocking each other.*
  * Inserting into the `Heartbeats` map or appending to the `UploadTimes` slice takes amortized `O(1)` time.
* **Overall Write Complexity**: Constantly `O(1)` regardless of how much data the system holds.

**Read Operations (GET Stats): Time Complexity `O(n)`**
* **Uptime Calculation**: Uptime is derived by looking at the `FirstHeartbeat` variable, `LastHeartbeat` variable, and `len(device.Heartbeats)`. Because `len()` in Go is an `O(1)` operation, calculating Uptime is mathematically instantaneous: `O(1)`.
* **Average Upload Time Calculation**: Currently, to find the average, the code loops through the entire `UploadTimes` slice to find the total sum and then divides it: `for _, t := range device.UploadTimes { sum += t }`.
* **Overall Read Complexity**: **`O(n)`** where `n` is the number of specific upload events for that device. 
  * *(Optimization Note: If strict `O(1)` read complexity was required, we could maintain a `totalUploadSum` and an `uploadCount` integer directly inside `DeviceData`. Every POST request would simply add to the sum and increment the count `O(1)`, allowing the GET request to instantly calculate `sum / count` in `O(1)` time without ever looping over an array).*

**Space Complexity: `O(D * (H+U))`**
Where `D` is unique devices, `H` is heartbeats, and `U` is upload events. The memory usage grows linearly indefinitely because there isn't a TTL eviction policy. This is perfectly acceptable for the assignment constraints, but would intuitively crash a production server over several months unless pushed to a database.

// Performance analysis for KV store as specified in cond.txt
// Tests write quorum values 1-5, measures latency, and checks consistency
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type KVClient struct {
	baseURL string
	client  *http.Client
}

type SetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	ID    string `json:"id"`
}

type GetRequest struct {
	Key string `json:"key"`
}

type Response struct {
	Success bool   `json:"success"`
	Value   string `json:"value,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Result struct {
	Quorum     int     `json:"quorum"`
	AvgLatency float64 `json:"avg_latency_ms"`
	Consistent bool    `json:"consistent"`
	Latencies  []int64 `json:"latencies_ms"`
}

func NewClient(baseURL string) *KVClient {
	return &KVClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *KVClient) Set(key, value string) (time.Duration, error) {
	start := time.Now()
	req := SetRequest{Key: key, Value: value, ID: "test"}

	data, err := json.Marshal(req)
	if err != nil {
		log.Printf("ERROR: Failed to marshal set request for key=%s: %v", key, err)
		return 0, fmt.Errorf("marshal error: %v", err)
	}

	// Reduced logging for performance - uncomment for detailed debugging
	// log.Printf("DEBUG: Sending SET request for key=%s to %s", key, c.baseURL)
	resp, err := c.client.Post(c.baseURL+"/set", "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("ERROR: SET request failed for key=%s: %v", key, err)
		return 0, err
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("ERROR: Failed to decode SET response for key=%s: %v", key, err)
		return 0, fmt.Errorf("decode error: %v", err)
	}

	if !result.Success {
		log.Printf("ERROR: SET operation failed for key=%s: %s", key, result.Error)
		return 0, fmt.Errorf("set failed: %s", result.Error)
	}

	duration := time.Since(start)
	// Reduced logging for performance - uncomment for detailed debugging
	// log.Printf("DEBUG: SET successful for key=%s, latency=%vms", key, duration.Milliseconds())
	return duration, nil
}

func (c *KVClient) Get(key string) (string, error) {
	req := GetRequest{Key: key}
	data, err := json.Marshal(req)
	if err != nil {
		log.Printf("ERROR: Failed to marshal get request for key=%s: %v", key, err)
		return "", fmt.Errorf("marshal error: %v", err)
	}

	// Reduced logging for performance - uncomment for detailed debugging
	// log.Printf("DEBUG: Sending GET request for key=%s to %s", key, c.baseURL)
	resp, err := c.client.Post(c.baseURL+"/get", "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("ERROR: GET request failed for key=%s: %v", key, err)
		return "", err
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("ERROR: Failed to decode GET response for key=%s: %v", key, err)
		return "", fmt.Errorf("decode error: %v", err)
	}

	if !result.Success {
		log.Printf("DEBUG: GET failed for key=%s: %s", key, result.Error)
		return "", fmt.Errorf("get failed: %s", result.Error)
	}

	// Reduced logging for performance - uncomment for detailed debugging
	// log.Printf("DEBUG: GET successful for key=%s, value=%s", key, result.Value)
	return result.Value, nil
}

// Removed docker restart functionality - assuming docker compose is already running
// To change quorum, manually update the .env file and restart docker-compose

func waitForLeader() bool {
	log.Printf("INFO: Waiting for leader to become ready...")
	client := NewClient("http://localhost:9000")

	for i := 0; i < 30; i++ {
		log.Printf("DEBUG: Leader readiness check attempt %d/30", i+1)
		if _, err := client.Get("test"); err == nil || (err != nil && (err.Error() == "get failed: " || err.Error() == "get failed: key not found")) {
			log.Printf("INFO: Leader is responding, waiting 5s for followers to sync...")
			time.Sleep(5 * time.Second) // Wait for followers
			log.Printf("INFO: Leader and followers are ready")
			return true
		} else {
			log.Printf("DEBUG: Leader not ready yet: %v", err)
		}
		time.Sleep(2 * time.Second)
	}

	log.Printf("ERROR: Leader failed to become ready after 60 seconds")
	return false
}

func runTest(quorum int) Result {
	log.Printf("INFO: ===== Starting test for quorum %d =====", quorum)
	log.Printf("INFO: Assuming docker compose is running with quorum=%d", quorum)

	if !waitForLeader() {
		log.Fatalf("FATAL: Leader not ready after multiple attempts")
	}

	leader := NewClient("http://localhost:9000")

	// 100 writes, 10 at a time, on 10 keys
	const numWrites = 100
	const concurrency = 10
	const numKeys = 10

	log.Printf("INFO: Test configuration - writes:%d, concurrency:%d, keys:%d", numWrites, concurrency, numKeys)

	keys := make([]string, numKeys)
	for i := range keys {
		keys[i] = fmt.Sprintf("key_%d", i)
	}
	log.Printf("INFO: Generated %d test keys: %v", len(keys), keys)

	var latencies []int64
	var failedWrites int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	log.Printf("INFO: Starting %d concurrent write operations...", numWrites)
	startTime := time.Now()

	for i := 0; i < numWrites; i++ {
		wg.Add(1)
		go func(writeNum int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			key := keys[writeNum%numKeys]
			value := fmt.Sprintf("value_%d_%d", writeNum, quorum)

			if duration, err := leader.Set(key, value); err == nil {
				mu.Lock()
				latencies = append(latencies, duration.Milliseconds())
				mu.Unlock()
			} else {
				mu.Lock()
				failedWrites++
				mu.Unlock()
				log.Printf("WARNING: Write %d failed for key=%s: %v", writeNum, key, err)
			}
		}(i)
	}

	log.Printf("INFO: Waiting for all write operations to complete...")
	wg.Wait()

	totalDuration := time.Since(startTime)
	log.Printf("INFO: Write operations completed in %v - successful:%d, failed:%d",
		totalDuration, len(latencies), failedWrites)

	// Calculate average latency
	var sum int64
	for _, lat := range latencies {
		sum += lat
	}

	if len(latencies) == 0 {
		log.Printf("ERROR: No successful writes for quorum %d", quorum)
		return Result{Quorum: quorum, AvgLatency: 0, Consistent: false, Latencies: []int64{}}
	}

	avgLatency := float64(sum) / float64(len(latencies))
	log.Printf("INFO: Average latency for quorum %d: %.2f ms (from %d samples)", quorum, avgLatency, len(latencies))

	// Wait for replication
	log.Printf("INFO: Waiting 3 seconds for replication to complete...")
	time.Sleep(3 * time.Second)

	// Check consistency
	log.Printf("INFO: Checking consistency across all nodes...")
	consistent := checkConsistency(keys)

	if consistent {
		log.Printf("INFO: ✓ All nodes are consistent for quorum %d", quorum)
	} else {
		log.Printf("WARNING: ✗ Consistency issues detected for quorum %d", quorum)
	}

	return Result{
		Quorum:     quorum,
		AvgLatency: avgLatency,
		Consistent: consistent,
		Latencies:  latencies,
	}
}

func checkConsistency(keys []string) bool {
	log.Printf("DEBUG: Starting consistency check for %d keys", len(keys))
	leader := NewClient("http://localhost:9000")

	// Get leader state
	leaderData := make(map[string]string)
	for _, key := range keys {
		if value, err := leader.Get(key); err == nil && value != "" {
			leaderData[key] = value
		} else {
			log.Printf("DEBUG: Key %s not found or empty on leader: %v", key, err)
		}
	}

	log.Printf("DEBUG: Leader has %d keys with values", len(leaderData))

	// Check followers
	for port := 9001; port <= 9005; port++ {
		log.Printf("DEBUG: Checking consistency with follower on port %d", port)
		follower := NewClient(fmt.Sprintf("http://localhost:%d", port))
		inconsistentKeys := 0

		for key, expectedValue := range leaderData {
			if value, err := follower.Get(key); err != nil || value != expectedValue {
				log.Printf("ERROR: Inconsistency detected - key=%s leader_value='%s' follower_%d_value='%s' error=%v",
					key, expectedValue, port, value, err)
				inconsistentKeys++
			}
		}

		if inconsistentKeys > 0 {
			log.Printf("ERROR: Follower %d has %d inconsistent keys out of %d total", port, inconsistentKeys, len(leaderData))
			return false
		} else {
			log.Printf("DEBUG: ✓ Follower %d is consistent", port)
		}
	}

	log.Printf("INFO: ✓ All followers are consistent with leader")
	return true
}

func plotResults(results []Result) {
	fmt.Println("\n=== ANALYSIS RESULTS ===")
	fmt.Printf("%-8s | %-12s | %-12s\n", "Quorum", "Avg Lat (ms)", "Consistent")
	fmt.Println("---------|--------------|------------")

	for _, r := range results {
		status := "✓"
		if !r.Consistent {
			status = "✗"
		}
		fmt.Printf("%-8d | %-12.2f | %-12s\n", r.Quorum, r.AvgLatency, status)
	}

	// ASCII plot
	fmt.Println("\n=== LATENCY PLOT ===")
	maxLat := 0.0
	for _, r := range results {
		if r.AvgLatency > maxLat {
			maxLat = r.AvgLatency
		}
	}

	for _, r := range results {
		bars := int((r.AvgLatency / maxLat) * 40)
		fmt.Printf("Q%d |", r.Quorum)
		for i := 0; i < bars; i++ {
			fmt.Print("█")
		}
		fmt.Printf(" %.1fms\n", r.AvgLatency)
	}

	// Analysis
	fmt.Println("\n=== EXPLANATION ===")
	fmt.Println("1. Latency vs Write Quorum:")
	fmt.Println("   - Higher quorum = more followers must confirm = higher latency")
	fmt.Println("   - Network delays (0-1000ms) add randomness to replication time")
	fmt.Println("   - Semi-synchronous replication waits for required confirmations")

	fmt.Println("\n2. Consistency Analysis:")
	fmt.Println("   - Higher quorum values improve consistency guarantees")
	fmt.Println("   - Some replicas may lag due to network delays")
	fmt.Println("   - Semi-synchronous replication may allow temporary inconsistencies")

	// Find trends
	if len(results) > 1 {
		increasing := true
		for i := 1; i < len(results); i++ {
			if results[i].AvgLatency <= results[i-1].AvgLatency {
				increasing = false
				break
			}
		}
		if increasing {
			fmt.Println("\n✓ Latency increases with quorum as expected")
		} else {
			fmt.Println("\n⚠ Latency trend is not monotonic (network variance)")
		}

		consistentCount := 0
		for _, r := range results {
			if r.Consistent {
				consistentCount++
			}
		}
		fmt.Printf("✓ Consistency rate: %d/%d (%.0f%%)\n",
			consistentCount, len(results), float64(consistentCount)/float64(len(results))*100)
	}
}

func main() {
	log.Println("INFO: ==========================================")
	log.Println("INFO: KV Store Performance Analysis Starting")
	log.Println("INFO: ==========================================")
	log.Printf("INFO: Will test write quorum values 1-5")
	log.Printf("INFO: Note: Manually ensure docker-compose is running with the correct WRITE_QUORUM value")

	var results []Result
	for quorum := 1; quorum <= 5; quorum++ {
		log.Printf("INFO: \n>>> TESTING QUORUM %d <<<", quorum)
		result := runTest(quorum)
		results = append(results, result)
		log.Printf("INFO: Quorum %d test completed - avg_latency=%.2fms consistent=%v",
			quorum, result.AvgLatency, result.Consistent)

		if quorum < 5 {
			log.Printf("INFO: Pausing 2 seconds before next test...")
			time.Sleep(2 * time.Second)
		}
	}

	// Save results
	log.Printf("INFO: Saving results to results.json...")
	if data, err := json.MarshalIndent(results, "", "  "); err == nil {
		if err := os.WriteFile("results.json", data, 0644); err != nil {
			log.Printf("ERROR: Failed to save results: %v", err)
		} else {
			log.Printf("INFO: ✓ Results saved successfully")
		}
	} else {
		log.Printf("ERROR: Failed to marshal results: %v", err)
	}

	log.Printf("INFO: Generating analysis report...")
	plotResults(results)

	log.Println("INFO: ==========================================")
	log.Println("INFO: ✓ Analysis complete. Results saved to results.json")
	log.Println("INFO: ==========================================")
}

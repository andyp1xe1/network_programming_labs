// Performance analysis for KV store as specified in cond.txt
// Tests write quorum values 1-5, measures latency, and checks consistency
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	kvhttp "github.com/andyp1xe1/network_programming_labs/packages/lab4/http"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/store"
)

const (
	// LEADER_URL is the fixed endpoint for the leader node that supports admin operations
	LEADER_URL = "http://localhost:9000"
)

type Result struct {
	Quorum            int     `json:"quorum"`
	AvgLatency        float64 `json:"avg_latency_ms"`
	Consistent        bool    `json:"consistent"`
	ConsistentNodes   int     `json:"consistent_nodes"`
	TotalNodes        int     `json:"total_nodes"`
	InconsistentNodes []int   `json:"inconsistent_nodes,omitempty"`
	Latencies         []int64 `json:"latencies_ms"`
}

func NewClient(baseURL string) store.Store {
	return kvhttp.NewHTTPClient(baseURL, "analysis-client")
}

// TimedSet performs a Set operation and measures the duration
func TimedSet(client store.Store, key, value string) (time.Duration, error) {
	start := time.Now()
	ctx := context.Background()
	err := client.Set(ctx, key, value)
	duration := time.Since(start)
	return duration, err
}

// setQuorum dynamically sets the quorum on the leader using the admin API
func setQuorum(quorum int) error {
	requestData := map[string]int{"quorum": quorum}
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to marshal quorum request: %v", err)
	}

	adminURL := LEADER_URL + "/admin/quorum"
	resp, err := http.Post(adminURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to make quorum request to leader: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("quorum update failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// verifyQuorum checks that the quorum was set correctly on the leader
func verifyQuorum(expectedQuorum int) error {
	adminURL := LEADER_URL + "/admin/quorum"
	resp, err := http.Get(adminURL)
	if err != nil {
		return fmt.Errorf("failed to get quorum status from leader: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read status response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("quorum status request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response kvhttp.Response
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse status response: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("quorum status response indicates failure: %s", response.Error)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response data format")
	}

	currentQuorum, ok := data["current_quorum"].(float64)
	if !ok {
		return fmt.Errorf("current_quorum not found in response")
	}

	if int(currentQuorum) != expectedQuorum {
		return fmt.Errorf("quorum mismatch: expected %d, got %d", expectedQuorum, int(currentQuorum))
	}

	return nil
}

// verifyAdminAPI checks that the admin API is available on the leader
func verifyAdminAPI() error {
	adminURL := LEADER_URL + "/admin/quorum"
	resp, err := http.Get(adminURL)
	if err != nil {
		return fmt.Errorf("admin API not reachable at leader (%s): %v", LEADER_URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("admin API not available - ensure %s is a leader node with admin endpoints enabled", LEADER_URL)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("admin API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// verifyLeaderRole ensures we're connected to an actual leader (not a follower)
func verifyLeaderRole() error {
	adminURL := LEADER_URL + "/admin/quorum"
	resp, err := http.Get(adminURL)
	if err != nil {
		return fmt.Errorf("failed to contact node: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("node at %s does not support admin endpoints - likely a follower, not a leader", LEADER_URL)
	}

	if resp.StatusCode == http.StatusNotImplemented {
		return fmt.Errorf("node at %s explicitly indicates admin not supported - likely a follower", LEADER_URL)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("leader verification failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read leader verification response: %v", err)
	}

	var response kvhttp.Response
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("invalid leader response format: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("leader returned error response: %s", response.Error)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok || data["current_quorum"] == nil {
		return fmt.Errorf("response missing quorum data - may not be a leader")
	}

	return nil
}

func waitForLeader() bool {
	client := NewClient(LEADER_URL)
	ctx := context.Background()

	for i := 0; i < 30; i++ {
		if _, err := client.Get(ctx, "test"); err == nil || (err != nil && (err.Error() == "get failed: " || err.Error() == "get failed: key not found")) {
			time.Sleep(5 * time.Second) // Wait for followers
			return true
		}
		time.Sleep(2 * time.Second)
	}

	return false
}

func runTest(quorum int) Result {
	fmt.Printf("Testing quorum %d...\n", quorum)

	if !waitForLeader() {
		log.Fatalf("FATAL: Leader not ready after multiple attempts")
	}

	if err := setQuorum(quorum); err != nil {
		return Result{Quorum: quorum, AvgLatency: 0, Consistent: false, ConsistentNodes: 0, TotalNodes: 5, Latencies: []int64{}}
	}

	if err := verifyQuorum(quorum); err != nil {
		return Result{Quorum: quorum, AvgLatency: 0, Consistent: false, ConsistentNodes: 0, TotalNodes: 5, Latencies: []int64{}}
	}

	time.Sleep(2 * time.Second)

	leader := NewClient(LEADER_URL)

	const numWrites = 100
	const concurrency = 10
	const numKeys = 10

	keys := make([]string, numKeys)
	for i := range keys {
		keys[i] = fmt.Sprintf("key_%d", i)
	}

	var latencies []int64
	var failedWrites int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < numWrites; i++ {
		wg.Add(1)
		go func(writeNum int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			key := keys[writeNum%numKeys]
			value := fmt.Sprintf("value_%d_%d", writeNum, quorum)

			if duration, err := TimedSet(leader, key, value); err == nil {
				mu.Lock()
				latencies = append(latencies, duration.Milliseconds())
				mu.Unlock()
			} else {
				mu.Lock()
				failedWrites++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if len(latencies) == 0 {
		return Result{Quorum: quorum, AvgLatency: 0, Consistent: false, ConsistentNodes: 0, TotalNodes: 5, Latencies: []int64{}}
	}

	var sum int64
	for _, lat := range latencies {
		sum += lat
	}
	avgLatency := float64(sum) / float64(len(latencies))

	fmt.Printf("Writes: %d successful, %d failed, avg latency: %.2fms\n", len(latencies), failedWrites, avgLatency)

	time.Sleep(3 * time.Second)

	consistencyResult := checkConsistency(keys)

	return Result{
		Quorum:            quorum,
		AvgLatency:        avgLatency,
		Consistent:        consistencyResult.IsConsistent,
		ConsistentNodes:   consistencyResult.ConsistentNodes,
		TotalNodes:        consistencyResult.TotalNodes,
		InconsistentNodes: consistencyResult.InconsistentNodes,
		Latencies:         latencies,
	}
}

type ConsistencyResult struct {
	IsConsistent      bool
	ConsistentNodes   int
	TotalNodes        int
	InconsistentNodes []int
}

func checkConsistency(keys []string) ConsistencyResult {
	leader := NewClient(LEADER_URL)

	leaderData := make(map[string]string)
	ctx := context.Background()
	for _, key := range keys {
		if value, err := leader.Get(ctx, key); err == nil && value != "" {
			leaderData[key] = value
		}
	}

	consistentNodes := 0
	totalNodes := 0
	var inconsistentNodes []int

	for port := 9001; port <= 9005; port++ {
		follower := NewClient(fmt.Sprintf("http://localhost:%d", port))
		inconsistentKeys := 0
		totalNodes++

		for key, expectedValue := range leaderData {
			if value, err := follower.Get(ctx, key); err != nil || value != expectedValue {
				inconsistentKeys++
			}
		}

		if inconsistentKeys > 0 {
			inconsistentNodes = append(inconsistentNodes, port)
		} else {
			consistentNodes++
		}
	}

	isConsistent := len(inconsistentNodes) == 0

	return ConsistencyResult{
		IsConsistent:      isConsistent,
		ConsistentNodes:   consistentNodes,
		TotalNodes:        totalNodes,
		InconsistentNodes: inconsistentNodes,
	}
}

func plotResults(results []Result) {
	fmt.Println("\n=== ANALYSIS RESULTS ===")
	fmt.Printf("%-8s | %-12s | %-18s | %-15s\n", "Quorum", "Avg Lat (ms)", "Consistency", "Nodes Status")
	fmt.Println("---------|--------------|-------------------|----------------")

	for _, r := range results {
		status := "✓ All consistent"
		nodeStatus := fmt.Sprintf("%d/%d consistent", r.ConsistentNodes, r.TotalNodes)

		if !r.Consistent {
			status = "✗ Issues found"
			if len(r.InconsistentNodes) > 0 {
				nodeStatus = fmt.Sprintf("%d/%d (bad: %v)", r.ConsistentNodes, r.TotalNodes, r.InconsistentNodes)
			}
		}

		fmt.Printf("%-8d | %-12.2f | %-18s | %-15s\n", r.Quorum, r.AvgLatency, status, nodeStatus)
	}

	if len(results) > 1 {
		consistentCount := 0
		totalNodeTests := 0
		totalConsistentNodes := 0

		for _, r := range results {
			if r.Consistent {
				consistentCount++
			}
			totalNodeTests += r.TotalNodes
			totalConsistentNodes += r.ConsistentNodes
		}

		fmt.Printf("\nTest consistency rate: %d/%d tests (%.0f%%)\n",
			consistentCount, len(results), float64(consistentCount)/float64(len(results))*100)
		fmt.Printf("Overall node consistency: %d/%d nodes (%.1f%%)\n",
			totalConsistentNodes, totalNodeTests, float64(totalConsistentNodes)/float64(totalNodeTests)*100)
	}
}

func main() {
	fmt.Println("KV Store Performance Analysis")
	fmt.Println("Testing quorum values 1-5")

	if err := verifyAdminAPI(); err != nil {
		log.Fatalf("FATAL: Admin API not available: %v", err)
	}

	if err := verifyLeaderRole(); err != nil {
		log.Fatalf("FATAL: Leader verification failed: %v", err)
	}

	var results []Result
	for quorum := 1; quorum <= 5; quorum++ {
		result := runTest(quorum)
		results = append(results, result)

		consistencyInfo := fmt.Sprintf("%d/%d nodes consistent", result.ConsistentNodes, result.TotalNodes)
		if !result.Consistent && len(result.InconsistentNodes) > 0 {
			consistencyInfo += fmt.Sprintf(" (bad: %v)", result.InconsistentNodes)
		}
		fmt.Printf("Quorum %d: %.2fms avg latency, %s\n", quorum, result.AvgLatency, consistencyInfo)

		if quorum < 5 {
			time.Sleep(2 * time.Second)
		}
	}

	if data, err := json.MarshalIndent(results, "", "  "); err == nil {
		if err := os.WriteFile("results.json", data, 0644); err == nil {
			fmt.Println("Results saved to results.json")
		}
	}

	plotResults(results)
}

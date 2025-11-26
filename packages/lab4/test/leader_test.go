package test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/common"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/leader"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/store"
)

// Mock follower for testing
type mockFollower struct {
	data map[string]string
	fail bool
}

func (m *mockFollower) Set(ctx context.Context, key, value string) error {
	if m.fail {
		return fmt.Errorf("simulated follower failure")
	}
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[key] = value
	return nil
}

func (m *mockFollower) Get(ctx context.Context, key string) (string, error) {
	if m.data == nil {
		return "", fmt.Errorf("key not found")
	}
	value, exists := m.data[key]
	if !exists {
		return "", fmt.Errorf("key not found")
	}
	return value, nil
}

func (m *mockFollower) Delete(ctx context.Context, key string) error {
	if m.fail {
		return fmt.Errorf("simulated follower failure")
	}
	if m.data == nil {
		return fmt.Errorf("key not found")
	}
	delete(m.data, key)
	return nil
}

func (m *mockFollower) Exists(ctx context.Context, key string) (bool, error) {
	if m.data == nil {
		return false, nil
	}
	_, exists := m.data[key]
	return exists, nil
}

func TestLeaderReplication(t *testing.T) {
	// Create a leader store with mock followers
	baseStore := store.NewMapStore()

	// Create 3 mock followers
	follower1 := &mockFollower{}
	follower2 := &mockFollower{}
	follower3 := &mockFollower{fail: true} // This one will "fail"

	followers := []store.Store{follower1, follower2, follower3}

	config := leader.LeaderConfig{
		CommitThreshold: 2, // Need 2 confirmations
		MaxDelay:        time.Millisecond * 10,
		MinDelay:        time.Millisecond * 1,
	}

	leaderStore := leader.NewLeaderStore(baseStore, config, followers, "test-leader")

	// Test set operation with context containing ID
	ctx := common.CtxWithID("test-leader")

	err := leaderStore.Set(ctx, "test-key", "test-value")
	if err != nil {
		t.Fatalf("Set operation failed: %v", err)
	}

	// Verify leader has the value
	value, err := leaderStore.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get operation failed: %v", err)
	}

	if value != "test-value" {
		t.Fatalf("Expected 'test-value', got '%s'", value)
	}

	// Verify at least 2 followers have the value (since commit threshold is 2)
	time.Sleep(50 * time.Millisecond) // Wait for async replication

	successfulReplicas := 0
	if strings.Compare(follower1.data["test-key"], "test-value") == 0 {
		successfulReplicas++
	}
	if strings.Compare(follower2.data["test-key"], "test-value") == 0 {
		successfulReplicas++
	}

	t.Logf("Successful replicas: %d (expected >= 2)", successfulReplicas)

	if successfulReplicas < 2 {
		t.Fatalf("Expected at least 2 successful replicas, got %d", successfulReplicas)
	}

	t.Log("Leader replication test passed!")
}

func TestLeaderReplicationQuorum(t *testing.T) {
	// Test with quorum of 3 but only 2 working followers
	baseStore := store.NewMapStore()

	follower1 := &mockFollower{}
	follower2 := &mockFollower{fail: true}
	follower3 := &mockFollower{fail: true}

	followers := []store.Store{follower1, follower2, follower3}

	config := leader.LeaderConfig{
		CommitThreshold: 3, // Need 3 confirmations but only 1 working
		MaxDelay:        time.Millisecond * 10,
		MinDelay:        time.Millisecond * 1,
	}

	leaderStore := leader.NewLeaderStore(baseStore, config, followers, "test-leader")

	ctx := common.CtxWithID("test-leader")

	// This should fail because we can't get 3 confirmations
	err := leaderStore.Set(ctx, "test-key", "test-value")
	t.Logf("Set operation result: %v", err)
	if err == nil {
		t.Fatal("Expected set operation to fail due to insufficient confirmations")
	}

	t.Logf("Quorum test passed - correctly failed with: %v", err)
}

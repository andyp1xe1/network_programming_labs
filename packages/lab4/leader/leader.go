/*
	Package leader provides a  key-value store to function as a leader in a leader-follower replication setup.

It replicates write operations to follower stores using
semi-synchronous replication with a configurable write quorum.
*/
package leader

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/common"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/store"
)

type LeaderConfig struct {
	CommitThreshold int

	MaxDelay time.Duration
	MinDelay time.Duration
}

type LeaderStore struct {
	store.Store
	config    LeaderConfig
	followers []store.Store
	leaderID  string
}

func NewLeaderStore(store store.Store, config LeaderConfig, followers []store.Store, leaderID string) *LeaderStore {
	return &LeaderStore{store, config, followers, leaderID}
}

func (kvs *LeaderStore) Set(ctx context.Context, key, value string) error {
	log.Printf("LeaderStore: Setting key=%s value=%s", key, value)
	if err := kvs.Store.Set(ctx, key, value); err != nil {
		log.Printf("LeaderStore: Set failed locally for key=%s value=%s: %v", key, value, err)
		return err
	}

	return kvs.replicateToFollowers(func(follower store.Store) error {
		ctx := common.CtxWithID(kvs.leaderID)
		return follower.Set(ctx, key, value)
	})
}

func (kvs *LeaderStore) Delete(ctx context.Context, key string) error {
	if err := kvs.Store.Delete(ctx, key); err != nil {
		log.Printf("LeaderStore: Delete failed locally for key=%s: %v", key, err)
		return err
	}

	return kvs.replicateToFollowers(func(follower store.Store) error {
		ctx := common.CtxWithID(kvs.leaderID)
		return follower.Delete(ctx, key)
	})
}

func (kvs *LeaderStore) randomDelay() time.Duration {
	num := rand.Int64N(kvs.config.MaxDelay.Milliseconds() - kvs.config.MinDelay.Milliseconds() + 1)
	delay := time.Duration(num+kvs.config.MinDelay.Milliseconds()) * time.Millisecond
	return delay
}

// replicateToFollowers implements semi-synchronous replication with configurable write quorum
func (kvs *LeaderStore) replicateToFollowers(operation func(store.Store) error) error {
	if len(kvs.followers) == 0 {
		return nil
	}

	resultCh := make(chan error, len(kvs.followers))

	for _, follower := range kvs.followers {
		go func(f store.Store) {
			delay := kvs.randomDelay()
			time.Sleep(delay)
			err := operation(f)
			resultCh <- err
		}(follower)
	}

	successCount := 0
	requiredConfirmations := min(kvs.config.CommitThreshold, len(kvs.followers))
	var errors []error
	for i := 0; i < len(kvs.followers); i++ {
		err := <-resultCh
		if err == nil {
			successCount++
			log.Printf("Replication to follower succeeded (%d/%d)", successCount, requiredConfirmations)
			if successCount >= requiredConfirmations {
				return nil
			}
		} else {
			log.Printf("Replication to follower failed: %v", err)
			errors = append(errors, err)
		}
	}
	var err error
	if len(errors) > 0 {
		err = errors[0]
	} else {
		err = fmt.Errorf("unknown error during replication")
	}
	return fmt.Errorf("failed to achieve write quorum: %v", err)
}

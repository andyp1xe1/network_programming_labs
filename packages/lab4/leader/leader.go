/*
	Package leader provides a  key-value store to function as a leader in a leader-follower replication setup.

It replicates write operations to follower stores using
semi-synchronous replication with a configurable write quorum.
*/
package leader

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
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
	configMu  sync.RWMutex
	followers []store.Store
	leaderID  string
}

func NewLeaderStore(store store.Store, config LeaderConfig, followers []store.Store, leaderID string) *LeaderStore {
	return &LeaderStore{
		Store:     store,
		config:    config,
		configMu:  sync.RWMutex{},
		followers: followers,
		leaderID:  leaderID,
	}
}

func (kvs *LeaderStore) Set(ctx context.Context, key, value string) error {
	log.Printf("LeaderStore: Setting key=%s value=%s", key, value)
	if err := kvs.Store.Set(ctx, key, value); err != nil {
		log.Printf("LeaderStore: Set failed locally for key=%s value=%s: %v", key, value, err)
		return err
	}

	vctx := common.CtxWithID(kvs.leaderID)
	if vs, ok := kvs.Store.(*store.VStore); ok {
		if meta, ok := vs.GetMeta(key); ok {
			vctx = common.CtxWithVersion(vctx, meta.Version)
		}
	}

	return kvs.replicateToFollowers(func(follower store.Store) error {
		return follower.Set(vctx, key, value)
	})
}

func (kvs *LeaderStore) Delete(ctx context.Context, key string) error {
	vctx := common.CtxWithID(kvs.leaderID)
	if vs, ok := kvs.Store.(*store.VStore); ok {
		if meta, ok := vs.GetMeta(key); ok {
			vctx = common.CtxWithVersion(vctx, meta.Version)
		}
	}

	if err := kvs.Store.Delete(ctx, key); err != nil {
		log.Printf("LeaderStore: Delete failed locally for key=%s: %v", key, err)
		return err
	}

	return kvs.replicateToFollowers(func(follower store.Store) error {
		return follower.Delete(vctx, key)
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
	kvs.configMu.RLock()
	requiredConfirmations := min(kvs.config.CommitThreshold, len(kvs.followers))
	kvs.configMu.RUnlock()
	var errs []error
	for i := 0; i < len(kvs.followers); i++ {
		err := <-resultCh
		if err == nil {
			successCount++
			log.Printf("Replication to follower succeeded (%d/%d)", successCount, requiredConfirmations)
			if successCount >= requiredConfirmations {
				return nil
			}
		} else if errors.Is(err, store.ErrorVersionConflict) {
			log.Printf("Replication to follower failed: %v", err)
			errs = append(errs, err)
		} else if errors.Is(err, store.ErrorGreaterVersionRequired) {
			log.Printf("Replication to follower dropped: %v", err)
		}
	}
	var err error
	if len(errs) > 0 {
		err = errs[0]
	} else {
		err = fmt.Errorf("unknown error during replication")
	}
	return fmt.Errorf("failed to achieve write quorum: %v", err)
}

// SetCommitThreshold updates the commit threshold (write quorum) dynamically
func (kvs *LeaderStore) SetCommitThreshold(threshold int) error {
	if threshold < 1 {
		return fmt.Errorf("commit threshold must be at least 1, got %d", threshold)
	}
	if threshold > len(kvs.followers) {
		return fmt.Errorf("commit threshold %d cannot exceed number of followers %d", threshold, len(kvs.followers))
	}

	kvs.configMu.Lock()
	oldThreshold := kvs.config.CommitThreshold
	kvs.config.CommitThreshold = threshold
	kvs.configMu.Unlock()

	log.Printf("LeaderStore: Updated commit threshold from %d to %d", oldThreshold, threshold)
	return nil
}

// GetCommitThreshold returns the current commit threshold
func (kvs *LeaderStore) GetCommitThreshold() int {
	kvs.configMu.RLock()
	threshold := kvs.config.CommitThreshold
	kvs.configMu.RUnlock()
	return threshold
}

// GetFollowerCount returns the number of configured followers
func (kvs *LeaderStore) GetFollowerCount() int {
	return len(kvs.followers)
}

// GetMeta returns the metadata for a key if the underlying store supports it
func (kvs *LeaderStore) GetMeta(key string) (store.Metadata, bool) {
	if metaStore, ok := kvs.Store.(store.MetadataStore); ok {
		return metaStore.GetMeta(key)
	}
	return store.Metadata{}, false
}

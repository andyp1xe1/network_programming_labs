/*
	Package follower implements a key-value store that only accepts write operations from a designated leader.

It wraps an existing key-value store and checks the context of incoming requests
to ensure they originate from the leader before allowing write operations.
Read operations are not restricted.
*/
package follower

import (
	"context"
	"fmt"
	"log"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/common"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/store"
)

type FollowerStore struct {
	store.Store
	ID       string
	leaderID string
}

func NewFollowerStore(store store.Store, id, lid string) *FollowerStore {
	return &FollowerStore{store, id, lid}
}

func (kvf *FollowerStore) checkLeader(ctx context.Context) error {
	id, ok := common.IDFromCtx(ctx)
	if !ok {
		return fmt.Errorf("no ID found in context")
	}
	if id != kvf.leaderID {
		return fmt.Errorf("request sent to follower from non-leader ID %s", id)
	}
	return nil
}

func (kvf *FollowerStore) Set(ctx context.Context, key, value string) error {
	if err := kvf.checkLeader(ctx); err != nil {
		return err
	}
	if err := kvf.Store.Set(ctx, key, value); err != nil {
		log.Printf("Follower %s: Failed to set key=%s value=%s: %v", kvf.ID, key, value, err)
		return err
	}
	log.Printf("Follower %s: Set key=%s value=%s", kvf.ID, key, value)
	return nil
}

func (kvf *FollowerStore) Delete(ctx context.Context, key string) error {
	if err := kvf.checkLeader(ctx); err != nil {
		return err
	}
	err := kvf.Store.Delete(ctx, key)
	if err != nil {
		log.Printf("Follower %s: Failed to delete key=%s: %v", kvf.ID, key, err)
		return err
	}
	log.Printf("Follower %s: Delete key=%s", kvf.ID, key)
	return nil
}

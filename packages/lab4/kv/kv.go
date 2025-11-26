/* Package kv provides the interface and configuration for a key-value store. */
package kv

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/follower"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/http"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/leader"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/rpc"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/store"
)

type KVserverConfig struct {
	RPCPort  string
	HTTPPort string

	ID string
}

type KVLeaderConfig struct {
	KVserverConfig
	FollowerIDs     []string
	CommitThreshold int
	MaxDelay        time.Duration
	MinDelay        time.Duration
}

type KVFollowerConfig struct {
	KVserverConfig
	LeaderID string
}

type KV struct {
	store.Store
	config KVserverConfig
}

func NewFollowerKV(config KVFollowerConfig) KV {
	f := follower.NewFollowerStore(store.NewMapStore(), config.ID, config.LeaderID)
	return KV{f, config.KVserverConfig}
}

func NewLeaderKV(config KVLeaderConfig) KV {
	followers := make([]store.Store, 0)
	for _, fid := range config.FollowerIDs {
		followerStore, error := rpc.NewKVrpcClient(fid)
		if error != nil {
			log.Fatal("Failed to connect to follower:", error)
		}
		followers = append(followers, followerStore)
	}

	lConf := leader.LeaderConfig{
		CommitThreshold: config.CommitThreshold,
		MaxDelay:        config.MaxDelay,
		MinDelay:        config.MinDelay,
	}
	l := leader.NewLeaderStore(store.NewMapStore(), lConf, followers, config.ID)
	return KV{l, config.KVserverConfig}
}

func (kv KV) Serve() {
	s := kv.Store
	rpcPort := kv.config.RPCPort
	httpPort := kv.config.HTTPPort
	id := kv.config.ID

	var enableHTTP bool
	if strings.Compare(httpPort, "") != 0 {
		enableHTTP = true
	}

	isLeader := false
	if _, ok := s.(*leader.LeaderStore); ok {
		isLeader = true
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		if err := rpc.ExposeRPC(s, rpcPort); err != nil {
			log.Fatal("Failed to start RPC server:", err)
		}
	})

	if enableHTTP {
		wg.Go(func() {
			log.Printf("Starting HTTP server (id=%s, isLeader=%t) on %s", id, isLeader, httpPort)
			if err := http.ExposeHTTP(s, httpPort); err != nil {
				log.Fatal("Failed to start HTTP server:", err)
			}
		})
	}

	log.Printf("KV store node started (id=%s, isLeader=%t)", id, isLeader)
	log.Printf("RPC server: %s, HTTP server: %s (enabled: %t)", rpcPort, httpPort, enableHTTP)

	wg.Wait()
}

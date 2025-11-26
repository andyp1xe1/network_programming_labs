package main

import (
	"flag"
	"strings"
	"time"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/kv"
)

func main() {
	id := flag.String("id", "", "Node ID (e.g., node1)")
	leaderID := flag.String("leader", "", "Leader node ID (e.g., node1)")
	followers := flag.String(
		"followers", "",
		"Comma-separated list of follower addresses (e.g., localhost:8001,localhost:8002)",
	)
	commitThreshold := flag.Int("commit-threshold", 1, "Quorum commit threshold")
	maxDelay := flag.Int("max-delay", 0, "Maximum network delay in milliseconds")
	minDelay := flag.Int("min-delay", 0, "Minimum network delay in milliseconds")
	rpcPort := flag.String("rpc-port", ":8000", "RPC server port")
	httpPort := flag.String("http-port", ":9000", "HTTP server port")
	flag.Parse()

	isLeader := strings.Compare(*id, *leaderID) == 0 || strings.Compare(*leaderID, "") == 0

	conf := kv.KVserverConfig{
		RPCPort:  *rpcPort,
		HTTPPort: *httpPort,
		ID:       *id,
	}

	var kvServer kv.KV
	if isLeader {
		var followerIDs []string
		if strings.Compare(*followers, "") != 0 {
			followerIDs = strings.Split(*followers, ",")
		}

		lConf := kv.KVLeaderConfig{
			KVserverConfig:  conf,
			FollowerIDs:     followerIDs,
			CommitThreshold: *commitThreshold,
			MaxDelay:        time.Duration(*maxDelay) * time.Millisecond,
			MinDelay:        time.Duration(*minDelay) * time.Millisecond,
		}
		kvServer = kv.NewLeaderKV(lConf)
	} else {
		fConf := kv.KVFollowerConfig{
			KVserverConfig: conf,
			LeaderID:       *leaderID,
		}
		kvServer = kv.NewFollowerKV(fConf)
	}

	kvServer.Serve()

}

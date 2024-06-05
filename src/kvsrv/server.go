package kvsrv

import (
	"log"
	"sync"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type KVServer struct {
	mu                 sync.Mutex
	kvStore            map[string]string
	logicalCounts      map[int64]int
	appendCacheResults map[int64]string
}

func (kv *KVServer) checkLogicCount(clientID int64, logicalCount int) bool {
	if kv.logicalCounts[clientID] <= logicalCount {
		return true
	}

	DPrintf("Duplicate request from Client ID %d, Logical count: %d.", clientID, logicalCount)
	return false
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	/* DPrintf("Report Recieved args: Key: %v, Client ID: %v, Logical count: %v", args.Key, args.ClientID, args.LogicalCount) */

	if !kv.checkLogicCount(args.ClientID, args.LogicalCount) {
		return
	}

	reply.Value = kv.kvStore[args.Key]
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	DPrintf("Recieve Put request args: Key: %v, Value: %v, Client ID: %v, Logical count: %v", args.Key, args.Value, args.ClientID, args.LogicalCount)

	if !kv.checkLogicCount(args.ClientID, args.LogicalCount) {
		return
	}

	if args.LogicalCount > kv.logicalCounts[args.ClientID] {
		kv.kvStore[args.Key] = args.Value
		kv.logicalCounts[args.ClientID] = args.LogicalCount
	}
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	DPrintf("Recieve Append request args: Key: %v, Value: %v, Client ID: %v, Logical count: %v", args.Key, args.Value, args.ClientID, args.LogicalCount)

	if !kv.checkLogicCount(args.ClientID, args.LogicalCount) {
		return
	}

	var old_value string
	if args.LogicalCount > kv.logicalCounts[args.ClientID] {
		old_value = kv.kvStore[args.Key]
		kv.appendCacheResults[args.ClientID] = old_value
		kv.kvStore[args.Key] = old_value + args.Value
		kv.logicalCounts[args.ClientID] = args.LogicalCount
	} else {
		old_value = kv.appendCacheResults[args.ClientID]
	}
	reply.Value = old_value
}

func StartKVServer() *KVServer {
	kv := new(KVServer)
	kv.mu = sync.Mutex{}
	kv.kvStore = make(map[string]string)
	kv.logicalCounts = make(map[int64]int)
	kv.appendCacheResults = make(map[int64]string)

	DPrintf("Initialized KVServer.")
	return kv
}

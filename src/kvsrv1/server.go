package kvsrv

import (
	"log"
	"sync"

	"6.5840/kvsrv1/rpc"
	"6.5840/labrpc"
	tester "6.5840/tester1"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type KVServer struct {
	mu           sync.Mutex
	kvStore      map[string]string
	versionStore map[string]rpc.Tversion
}

func MakeKVServer() *KVServer {
	kv := &KVServer{
		mu:           sync.Mutex{},
		kvStore:      make(map[string]string),
		versionStore: make(map[string]rpc.Tversion),
	}
	return kv
}

// Get returns the value and version for args.Key, if args.Key
// exists. Otherwise, Get returns ErrNoKey.
func (kv *KVServer) Get(args *rpc.GetArgs, reply *rpc.GetReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if value, ok := kv.kvStore[args.Key]; ok {
		reply.Value = value
		reply.Version = kv.versionStore[args.Key]
		reply.Err = rpc.OK
		DPrintf("Server: Get key: %s value: %s with version %d", args.Key, value, reply.Version)
	} else {
		reply.Err = rpc.ErrNoKey
		DPrintf("Server: Get key: %s not found", args.Key)
	}
}

// Update the value for a key if args.Version matches the version of
// the key on the server. If versions don't match, return ErrVersion.
// If the key doesn't exist, Put installs the value if the
// args.Version is 0, and returns ErrNoKey otherwise.
func (kv *KVServer) Put(args *rpc.PutArgs, reply *rpc.PutReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if _, ok := kv.kvStore[args.Key]; ok {
		// Key exists.
		version := kv.versionStore[args.Key]
		if args.Version != version {
			DPrintf("Server: Put version mismatch key: %s expected version %d got %d", args.Key, version, args.Version)
			reply.Err = rpc.ErrVersion
		} else {
			if args.Value == kv.kvStore[args.Key] {
				// No change in value.
				DPrintf("Server: Put no change in value %s: %s", args.Key, args.Value)
			} else {
				// Replace the value.
				kv.kvStore[args.Key] = args.Value
				DPrintf("Server: Put replace key: %s with new value: %s with version %d", args.Key, args.Value, version)
			}
			kv.versionStore[args.Key]++
			reply.Err = rpc.OK
		}
	} else {
		// Key doesn't exist.
		if args.Version == 0 {
			// Install the value if the version is 0.
			kv.kvStore[args.Key] = args.Value
			kv.versionStore[args.Key] = 1
			reply.Err = rpc.OK
			DPrintf("Server: Put install new key: %s value: %s", args.Key, args.Value)
		} else {
			reply.Err = rpc.ErrNoKey
		}
	}
}

// You can ignore Kill() for this lab
func (kv *KVServer) Kill() {
}

// You can ignore all arguments; they are for replicated KVservers
func StartKVServer(ends []*labrpc.ClientEnd, gid tester.Tgid, srv int, persister *tester.Persister) []tester.IService {
	kv := MakeKVServer()
	return []tester.IService{kv}
}

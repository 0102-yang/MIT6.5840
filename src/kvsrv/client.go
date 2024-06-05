package kvsrv

import (
	"crypto/rand"
	"math/big"
	"time"

	"6.5840/labrpc"
)

type Clerk struct {
	server       *labrpc.ClientEnd
	clientID     int64
	logicalCount int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(server *labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.server = server
	ck.clientID = nrand()
	ck.logicalCount = 1
	DPrintf("Initialized Client-Clerk ID: %v. Sign in to server.", ck.clientID)
	return ck
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string) string {
	getArgs := &GetArgs{
		Key:          key,
		ClientID:     ck.clientID,
		LogicalCount: ck.logicalCount,
	}
	getReply := &GetReply{}

	for {
		ok := ck.server.Call("KVServer.Get", getArgs, getReply)
		if ok {
			ck.logicalCount++
			return getReply.Value
		}
		time.Sleep(time.Duration(10 * time.Millisecond))
	}
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) string {
	putAppendArgs := &PutAppendArgs{
		Key:          key,
		Value:        value,
		ClientID:     ck.clientID,
		LogicalCount: ck.logicalCount,
	}
	putAppendReply := &PutAppendReply{}
	method := "KVServer." + op

	for {
		ok := ck.server.Call(method, putAppendArgs, putAppendReply)
		if ok {
			ck.logicalCount++
			if op == "Append" {
				return putAppendReply.Value
			} else {
				return ""
			}
		}
		time.Sleep(time.Duration(10 * time.Millisecond))
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

// Append value to key's value and return that value
func (ck *Clerk) Append(key string, value string) string {
	return ck.PutAppend(key, value, "Append")
}

# MIT 6.8240 Distributed Systems - Go Implementation

This repository contains Go implementations of labs and projects for the MIT 6.8240 Distributed Systems course.
For more details about the course, visit the [course website](https://pdos.csail.mit.edu/6.824/).

## Lab 1: MapReduce

#### Task: Implement MapReduce system (moderate/hard)

In this lab, you will build a MapReduce system. The system includes:

- A worker process that:
  - Calls application Map and Reduce functions
  - Handles reading and writing files
- A coordinator process that:
  - Distributes tasks to workers
  - Handles failed workers

This lab is similar to the system described in the MapReduce paper. Note that this lab uses the term "coordinator" instead of the paper's "master".

#### Result:

<img src="images/Lab1 Test Result.png" alt="Lab1 Test Result" width="600">

## Lab2: Key/Value Server

### Introduction

In this lab you will build a key/value server for a single machine that ensures that each Put operation is executed at-most-once despite network failures and that the operations are linearizable. You will use this KV server to implement a lock. Later labs will replicate a server like this one to handle server crashes.

### Core concepts

#### KV Server

The KV server allows clients to interact using a Clerk, which sends RPCs to the server. Clients can perform two types of RPCs: Put(key, value, version) and Get(key).

- ​Put Operation
  This operation installs or replaces the value for a specific key in the server's in-memory map only if the provided version number matches the   server's current version number for that key. If the version numbers match, the server increments the version number. If the version numbers don't match, the server returns rpc.ErrVersion. A new key can be created by calling Put with a version number of 0, resulting in the server storing version 1.
- Get Operation
  This operation fetches the current value and its associated version for a given key. If the key does not exist on the server, the server returns rpc.ErrNoKey.

#### Versioning and Linearizability

Each key in the server's map is associated with a version number that records how many times the key has been written.
Maintaining version numbers is useful for implementing locks and ensuring at-most-once semantics for Put operations, especially in unreliable networks where clients might retransmit requests.
Linearizability:

Once the lab is completed and all tests are passed, the key/value service will provide linearizability from the client's perspective. This means that if client operations are not concurrent, each client's Clerk.Get and Clerk.Put will observe the modifications implied by the preceding sequence of operations.

For concurrent operations, the return values and final state will be the same as if the operations had executed one at a time in some order. Operations are considered concurrent if they overlap in time.
Linearizability is beneficial for applications because it ensures that the behavior is consistent with a single server processing requests sequentially, making it easier to reason about the system's state.

### Task A: Key/value server with reliable network (easy)

#### Task

Your first task is to implement a solution that works when there are no dropped messages. You'll need to add RPC-sending code to the Clerk Put/Get methods in client.go, and implement Put and Get RPC handlers in server.go.

#### Result

<img src="images/Lab2 Task A Result.png" alt="Task A Result" width="600">

### Task B: Implementing a lock using key/value clerk (moderate)

#### Task

In many distributed applications, clients running on different machines use a key/value server to coordinate their activities. For example, ZooKeeper and Etcd allow clients to coordinate using a distributed lock, in analogy with how threads in a Go program can coordinate with locks (i.e., sync.Mutex). Zookeeper and Etcd implement such a lock with conditional put.

In this exercise your task is to implement a lock layered on client Clerk.Put and Clerk.Get calls. The lock supports two methods: Acquire and Release. The lock's specification is that only one client can successfully acquire the lock at a time; other clients must wait until the first client has released the lock using Release.

We supply you with skeleton code and tests in src/kvsrv1/lock/. You will need to modify src/kvsrv1/lock/lock.go. Your Acquire and Release code can talk to your key/value server by calling lk.ck.Put() and lk.ck.Get().

If a client crashes while holding a lock, the lock will never be released. In a design more sophisticated than this lab, the client would attach a lease to a lock. When the lease expires, the lock server would release the lock on behalf of the client. In this lab clients don't crash and you can ignore this problem.

Implement Acquire and Release.

#### Result

<img src="images/Lab2 Task B Result.png" alt="Task B Result" width="600">

### Task C: Key/value server with dropped messages (moderate)

#### Task

The main challenge in this exercise is that the network may re-order, delay, or discard RPC requests and/or replies. To recover from discarded requests/replies, the Clerk must keep re-trying each RPC until it receives a reply from the server.

If the network discards an RPC request message, then the client re-sending the request will solve the problem: the server will receive and execute just the re-sent request.

However, the network might instead discard an RPC reply message. The client does not know which message was discarded; the client only observes that it received no reply. If it was the reply that was discarded, and the client re-sends the RPC request, then the server will receive two copies of the request. That's OK for a Get, since Get doesn't modify the server state. It is safe to resend a Put RPC with the same version number, since the server executes Put conditionally on the version number; if the server received and executed a Put RPC, it will respond to a re-transmitted copy of that RPC with rpc.ErrVersion rather than executing the Put a second time.

A tricky case is if the server replies with an rpc.ErrVersion in a response to an RPC that the Clerk retried. In this case, the Clerk cannot know if the Clerk's Put was executed by the server or not: the first RPC might have been executed by the server but the network may have discarded the successful response from the server, so that the server sent rpc.ErrVersion only for the retransmitted RPC. Or, it might be that another Clerk updated the key before the Clerk's first RPC arrived at the server, so that the server executed neither of the Clerk's RPCs and replied rpc.ErrVersion to both. Therefore, if a Clerk receives rpc.ErrVersion for a retransmitted Put RPC, Clerk.Put must return rpc.ErrMaybe to the application instead of rpc.ErrVersion since the request may have been executed. It is then up to the application to handle this case. If the server responds to an initial (not retransmitted) Put RPC with rpc.ErrVersion, then the Clerk should return rpc.ErrVersion to the application, since the RPC was definitely not executed by the server.

It would be more convenient for application developers if Put's were exactly-once (i.e., no rpc.ErrMaybe errors) but that is difficult to guarantee without maintaining state at the server for each Clerk. In the last exercise of this lab, you will implement a lock using your Clerk to explore how to program with at-most-once Clerk.Put.

Now you should modify your kvsrv1/client.go to continue in the face of dropped RPC requests and replies. A return value of true from the client's ck.clnt.Call() indicates that the client received an RPC reply from the server; a return value of false indicates that it did not receive a reply (more precisely, Call() waits for a reply message for a timeout interval, and returns false if no reply arrives within that time). Your Clerk should keep re-sending an RPC until it receives a reply. Keep in mind the discussion of rpc.ErrMaybe above. Your solution shouldn't require any changes to the server.

Add code to Clerk to retry if doesn't receive a reply.

#### Result

<img src="images/Lab2 Task C Result.png" alt="Task C Result" width="600">

### Task D: Implementing a lock using key/value clerk and unreliable network (easy)

#### Task

Modify your lock implementation to work correctly with your modified key/value client when the network is not reliable.

#### Result

<img src="images/Lab2 Task D Result.png" alt="Task D Result" width="600">

## Lab3 Raft

This is the first lab in a series where we build a fault-tolerant key/value storage system. In this lab, we implement Raft, a replicated state machine protocol. Future labs will build a key/value service on top of Raft and shard the service for higher performance.

### Replicated Service

A replicated service stores complete copies of its state on multiple servers for fault tolerance. This allows the service to operate despite server failures. However, failures may cause replicas to hold differing data copies.

### Raft Protocol

Raft organizes client requests into a log sequence, ensuring all replicas see the same log. Each replica executes client requests in log order, maintaining identical service state. If a server recovers from failure, Raft updates its log. Raft operates as long as a majority of servers are alive and communicative.

### Lab Goals
In this lab, implement Raft as a Go object type with methods, to be used as a module in a larger service. Raft instances communicate via RPC to maintain replicated logs. Your Raft interface will support an indefinite sequence of numbered log entries. Once a log entry is committed, Raft sends it to the larger service for execution.

Follow the design in the extended Raft paper, particularly Figure 2. Implement most of the paper's content, including saving persistent state and reading it after a node restarts. Do not implement cluster membership changes (Section 6).

This lab is due in four parts, each with a corresponding due date.
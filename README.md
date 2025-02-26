# MIT 6.8240 Distributed Systems - Go Implementation

This repository contains Go implementations of labs and projects for the MIT 6.8240 Distributed Systems course.
For more details about the course, visit the [course website](https://pdos.csail.mit.edu/6.824/).

## Lab 1: MapReduce

### Task

In this lab, you will build a MapReduce system. The system includes:

- A worker process that:
  - Calls application Map and Reduce functions
  - Handles reading and writing files
- A coordinator process that:
  - Distributes tasks to workers
  - Handles failed workers

This lab is similar to the system described in the MapReduce paper. Note that this lab uses the term "coordinator" instead of the paper's "master".

### Lab1 Test Result:
<img src="images/Lab1 Test Result.png" alt="Lab1 Test Result" width="600">

## Lab2: Key/Value Server

### Introduction

In this lab you will build a key/value server for a single machine that ensures that each Put operation is executed at-most-once despite network failures and that the operations are linearizable. You will use this KV server to implement a lock. Later labs will replicate a server like this one to handle server crashes.

### Core concepts

#### KV Server

The KV server allows clients to interact using a Clerk, which sends RPCs to the server. Clients can perform two types of RPCs: Put(key, value, version) and Get(key).

#### ​Put Operation
This operation installs or replaces the value for a specific key in the server's in-memory map only if the provided version number matches the server's current version number for that key. If the version numbers match, the server increments the version number. If the version numbers don't match, the server returns rpc.ErrVersion. A new key can be created by calling Put with a version number of 0, resulting in the server storing version 1.

#### Get Operation

This operation fetches the current value and its associated version for a given key. If the key does not exist on the server, the server returns rpc.ErrNoKey.

#### Versioning and Linearizability

Each key in the server's map is associated with a version number that records how many times the key has been written.
Maintaining version numbers is useful for implementing locks and ensuring at-most-once semantics for Put operations, especially in unreliable networks where clients might retransmit requests.
Linearizability:

Once the lab is completed and all tests are passed, the key/value service will provide linearizability from the client's perspective. This means that if client operations are not concurrent, each client's Clerk.Get and Clerk.Put will observe the modifications implied by the preceding sequence of operations.

For concurrent operations, the return values and final state will be the same as if the operations had executed one at a time in some order. Operations are considered concurrent if they overlap in time.
Linearizability is beneficial for applications because it ensures that the behavior is consistent with a single server processing requests sequentially, making it easier to reason about the system's state.

### Task A: Key/value server with reliable network(easy)

#### Target

Your first task is to implement a solution that works when there are no dropped messages. You'll need to add RPC-sending code to the Clerk Put/Get methods in client.go, and implement Put and Get RPC handlers in server.go.

#### Result

<img src="images/Lab2 Task A Result.png" alt="Task A Result" width="600">

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
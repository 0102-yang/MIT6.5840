# MIT 6.8240 Distributed Systems - Go Implementation

This repository contains Go implementations of labs and projects for the MIT 6.8240 Distributed Systems course.

# Course Information

For more details about the course, visit the [course website](https://pdos.csail.mit.edu/6.824/).

# Labs

## Lab 1: MapReduce

In this lab, you will build a MapReduce system. The system includes:

- A worker process that:
  - Calls application Map and Reduce functions
  - Handles reading and writing files
- A coordinator process that:
  - Distributes tasks to workers
  - Handles failed workers

This lab is similar to the system described in the MapReduce paper. Note that this lab uses the term "coordinator" instead of the paper's "master".

### Lab1 Test Result:
![Lab1 Test Result](images/Lab1%20Test%20Result.png)

## Lab2: Key/Value Server

In this lab, you will build a key/value server for a single machine. The server ensures that each operation is executed exactly once despite network failures and that the operations are linearizable. Later labs will replicate a server like this one to handle server crashes.

### Server Operations

Clients can send three different RPCs to the key/value server:

1. `Put(key, value)`: Installs or replaces the value for a particular key in the map.
2. `Append(key, arg)`: Appends `arg` to `key`'s value and returns the old value.
3. `Get(key)`: Fetches the current value for the key.

The server maintains an in-memory map of key/value pairs. Keys and values are strings. A `Get` for a non-existent key should return an empty string. An `Append` to a non-existent key should act as if the existing value were a zero-length string.

### Client-Server Interaction

Each client talks to the server through a `Clerk` with `Put/Append/Get` methods. A `Clerk` manages RPC interactions with the server.

Your server must arrange that application calls to `Clerk Get/Put/Append` methods be linearizable. If client requests aren't concurrent, each client `Get/Put/Append` call should observe the modifications to the state implied by the preceding sequence of calls.

For concurrent calls, the return values and final state must be the same as if the operations had executed one at a time in some order. Calls are concurrent if they overlap in time.

### Linearizability

Linearizability is convenient for applications because it's the behavior you'd see from a single server that processes requests one at a time. For example, if one client gets a successful response from the server for an update request, subsequently launched reads from other clients are guaranteed to see the effects of that update. Providing linearizability is relatively easy for a single server.

### Lab2 Test Result:
![Lab2 Test Result](images/Lab2%20Test%20Result.png)

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
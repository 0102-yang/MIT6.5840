# MIT 6.8240 Distributed Systems - Go Implementation

This repository contains Go implementations of labs and projects for the MIT 6.8240 Distributed Systems course.

## Course Information

For more details about the course, visit the [course website](https://pdos.csail.mit.edu/6.824/).

## Labs

### Lab 1: MapReduce

In this lab, you will build a MapReduce system. The system includes:

- A worker process that:
  - Calls application Map and Reduce functions
  - Handles reading and writing files
- A coordinator process that:
  - Distributes tasks to workers
  - Handles failed workers

This lab is similar to the system described in the MapReduce paper. Note that this lab uses the term "coordinator" instead of the paper's "master".
package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.
type RequestStatus int
type TaskType int

const (
	NotReady RequestStatus = iota
	Ready
	Completed
)
const (
	MapTask TaskType = iota
	ReduceTask
)

type RequestTaskArgs struct {
}
type RequestTaskReply struct {
	Status  RequestStatus
	Type    TaskType
	ID      int
	NMap    int
	NReduce int

	// For MapTask.
	MapFileName string
}

type ReportTaskArgs struct {
	ID   int
	Type TaskType
}
type ReportTaskReply struct {
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}

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

// Add your RPC definitions here.
type MapTaskArgs struct {
}

type MapTaskReply struct {
	Valid        bool
	MapTaskIndex int
	MapFilename  string
	NReduce      int
}

type ReduceTaskArgs struct {
}

type ReduceTaskStatusType int

const (
	NotReady ReduceTaskStatusType = iota
	Ready
	Invalid
)

type ReduceTaskReply struct {
	ReduceTaskStatus ReduceTaskStatusType
	ReduceTaskIndex  int
	NMap             int
}

type TaskDoneArgs struct {
	GlobalTaskNum int
}

type TaskDoneReply struct {
}

type GlobalTaskNumArgs struct {
	ReduceTaskNum int
}

type GlobalTaskNumReply struct {
	GlobalTaskNum int
}

type SignInWorkerArgs struct {
}

type SignInWorkerReply struct {
}

type SignOutWorkerArgs struct {
}

type SignOutWorkerReply struct {
}

type PermitWorkerExitArgs struct {
}

type PermitWorkerExitReply struct {
	CanExit bool
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

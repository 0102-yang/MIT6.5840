package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type MapTask struct {
	MapTaskIndex int
	Filename     string
}

type ReduceTask struct {
	ReduceTaskIndex int
}

type Coordinator struct {
	// Your definitions here.
	MapTasks          []MapTask
	ReduceTasks       []ReduceTask
	ReadySet          map[int]bool
	ProcessingSet     map[int]bool
	DoneSet           map[int]bool
	CurrentWorkersNum int
	NMap              int
	NReduce           int
	Mutex             sync.Mutex
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) handle_timeout(globalTaskIndex int) {
	time.AfterFunc(10*time.Second, func() {
		c.Mutex.Lock()
		defer c.Mutex.Unlock()

		_, exists := c.ProcessingSet[globalTaskIndex]
		if exists {
			delete(c.ProcessingSet, globalTaskIndex)
			c.ReadySet[globalTaskIndex] = true
			log.Printf("Coordinator: Global Task %v was timeout.\n", globalTaskIndex)
		}
	})
}

func (c *Coordinator) getGlobalTaskIndexByReduceTaskIndex(reduceTaskIndex int) int {
	return reduceTaskIndex + c.NMap
}

func (c *Coordinator) getReduceTaskIndexByGlobalTaskIndex(globalTaskIndex int) int {
	return globalTaskIndex - c.NMap
}

func (c *Coordinator) isMapTaskIndex(globalTaskIndex int) bool {
	return globalTaskIndex < c.NMap
}

func (c *Coordinator) isReduceTaskIndex(globalTaskIndex int) bool {
	return globalTaskIndex >= c.NMap
}

func (c *Coordinator) mapTasksAllDone() bool {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	for globalTaskIndex := range c.ReadySet {
		if c.isMapTaskIndex(globalTaskIndex) {
			return false
		}
	}
	for globalTaskIndex := range c.ProcessingSet {
		if c.isMapTaskIndex(globalTaskIndex) {
			return false
		}
	}

	return true
}

func (c *Coordinator) GetGlobalTaskNum(args *GlobalTaskNumArgs, reply *GlobalTaskNumReply) error {
	reply.GlobalTaskNum = c.getGlobalTaskIndexByReduceTaskIndex(args.ReduceTaskNum)
	return nil
}

func (c *Coordinator) GetMapTask(args *MapTaskArgs, reply *MapTaskReply) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	for globalTaskIndex := range c.ReadySet {
		if c.isMapTaskIndex(globalTaskIndex) {
			reply.Valid = true
			reply.MapTaskIndex = globalTaskIndex
			reply.MapFilename = c.MapTasks[globalTaskIndex].Filename
			reply.NReduce = c.NReduce

			delete(c.ReadySet, globalTaskIndex)
			c.ProcessingSet[globalTaskIndex] = true

			go c.handle_timeout(globalTaskIndex)
			return nil
		}
	}

	reply.Valid = false
	return nil
}

func (c *Coordinator) GetReduceTask(args *ReduceTaskArgs, reply *ReduceTaskReply) error {
	if !c.mapTasksAllDone() {
		reply.ReduceTaskStatus = NotReady
		return nil
	}

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	for globalTaskIndex := range c.ReadySet {
		if c.isReduceTaskIndex(globalTaskIndex) {
			reduceTaskIndex := c.getReduceTaskIndexByGlobalTaskIndex(globalTaskIndex)

			reply.ReduceTaskStatus = Ready
			reply.ReduceTaskIndex = reduceTaskIndex
			reply.NMap = c.NMap

			delete(c.ReadySet, globalTaskIndex)
			c.ProcessingSet[globalTaskIndex] = true

			go c.handle_timeout(globalTaskIndex)
			return nil
		}
	}

	reply.ReduceTaskStatus = Invalid
	return nil
}

func (c *Coordinator) TaskDone(args *TaskDoneArgs, reply *TaskDoneReply) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	_, exists := c.ProcessingSet[args.GlobalTaskNum]
	if exists {
		delete(c.ProcessingSet, args.GlobalTaskNum)

		if c.isMapTaskIndex(args.GlobalTaskNum) {
			log.Printf("Coordinator: Global Task %v (Map Task %v) is done successfully.\n", args.GlobalTaskNum, args.GlobalTaskNum)
		} else {
			log.Printf("Coordinator: Global Task %v (Reduce Task %v) is done successfully.\n", args.GlobalTaskNum, c.getReduceTaskIndexByGlobalTaskIndex(args.GlobalTaskNum))
		}
		c.DoneSet[args.GlobalTaskNum] = true
	}
	return nil
}

func (c *Coordinator) SignInWorker(args *SignInWorkerArgs, reply *SignInWorkerReply) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	log.Printf("Coordinator: A worker signed in.\n")

	c.CurrentWorkersNum++
	return nil
}

func (c *Coordinator) PermitWorkerExit(args *PermitWorkerExitArgs, reply *PermitWorkerExitReply) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	reply.CanExit = len(c.DoneSet) == c.NMap+c.NReduce
	return nil
}

func (c *Coordinator) SignOutWorker(args *SignOutWorkerArgs, reply *SignOutWorkerReply) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	log.Printf("Coordinator: A worker signed out.\n")

	c.CurrentWorkersNum--
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	return len(c.DoneSet) == c.NMap+c.NReduce && c.CurrentWorkersNum == 0
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	// Initialize the coordinator.
	c := Coordinator{
		MapTasks:          []MapTask{},
		ReduceTasks:       []ReduceTask{},
		ReadySet:          map[int]bool{},
		ProcessingSet:     map[int]bool{},
		DoneSet:           map[int]bool{},
		CurrentWorkersNum: 0,
		NMap:              len(files),
		NReduce:           nReduce,
		Mutex:             sync.Mutex{},
	}

	// Allocate tasks.
	for i, file := range files {
		c.MapTasks = append(c.MapTasks, MapTask{
			MapTaskIndex: i,
			Filename:     file,
		})
	}
	for i := 0; i < nReduce; i++ {
		c.ReduceTasks = append(c.ReduceTasks, ReduceTask{
			ReduceTaskIndex: i,
		})
	}

	// Allocate ready set.
	for i := 0; i < c.NMap; i++ {
		c.ReadySet[i] = true
	}
	for i := 0; i < nReduce; i++ {
		c.ReadySet[i+c.NMap] = true
	}

	log.Println("Coordinator initialized.")

	// Start the RPC server.
	c.server()
	return &c
}

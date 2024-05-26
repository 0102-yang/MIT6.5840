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
	mapTasks          []MapTask
	reduceTasks       []ReduceTask
	readySet          map[int]bool
	processingSet     map[int]bool
	doneSet           map[int]bool
	currentWorkersNum int
	nMap              int
	nReduce           int
	mutex             sync.Mutex
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) handle_timeout(globalTaskIndex int) {
	time.AfterFunc(10*time.Second, func() {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		_, exists := c.processingSet[globalTaskIndex]
		if exists {
			delete(c.processingSet, globalTaskIndex)
			c.readySet[globalTaskIndex] = true
			log.Printf("Coordinator: Global Task %v was timeout.\n", globalTaskIndex)
		}
	})
}

func (c *Coordinator) getGlobalTaskIndexByReduceTaskIndex(reduceTaskIndex int) int {
	return reduceTaskIndex + c.nMap
}

func (c *Coordinator) getReduceTaskIndexByGlobalTaskIndex(globalTaskIndex int) int {
	return globalTaskIndex - c.nMap
}

func (c *Coordinator) isMapTaskIndex(globalTaskIndex int) bool {
	return globalTaskIndex < c.nMap
}

func (c *Coordinator) isReduceTaskIndex(globalTaskIndex int) bool {
	return globalTaskIndex >= c.nMap
}

func (c *Coordinator) mapTasksAllDone() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for globalTaskIndex := range c.readySet {
		if c.isMapTaskIndex(globalTaskIndex) {
			return false
		}
	}
	for globalTaskIndex := range c.processingSet {
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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for globalTaskIndex := range c.readySet {
		if c.isMapTaskIndex(globalTaskIndex) {
			reply.Valid = true
			reply.MapTaskIndex = globalTaskIndex
			reply.MapFilename = c.mapTasks[globalTaskIndex].Filename
			reply.NReduce = c.nReduce

			delete(c.readySet, globalTaskIndex)
			c.processingSet[globalTaskIndex] = true

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

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for globalTaskIndex := range c.readySet {
		if c.isReduceTaskIndex(globalTaskIndex) {
			reduceTaskIndex := c.getReduceTaskIndexByGlobalTaskIndex(globalTaskIndex)

			reply.ReduceTaskStatus = Ready
			reply.ReduceTaskIndex = reduceTaskIndex
			reply.NMap = c.nMap

			delete(c.readySet, globalTaskIndex)
			c.processingSet[globalTaskIndex] = true

			go c.handle_timeout(globalTaskIndex)
			return nil
		}
	}

	reply.ReduceTaskStatus = Invalid
	return nil
}

func (c *Coordinator) TaskDone(args *TaskDoneArgs, reply *TaskDoneReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	_, exists := c.processingSet[args.GlobalTaskNum]
	if exists {
		delete(c.processingSet, args.GlobalTaskNum)

		if c.isMapTaskIndex(args.GlobalTaskNum) {
			log.Printf("Coordinator: Global Task %v (Map Task %v) is done successfully.\n", args.GlobalTaskNum, args.GlobalTaskNum)
		} else {
			log.Printf("Coordinator: Global Task %v (Reduce Task %v) is done successfully.\n", args.GlobalTaskNum, c.getReduceTaskIndexByGlobalTaskIndex(args.GlobalTaskNum))
		}
		c.doneSet[args.GlobalTaskNum] = true
	}
	return nil
}

func (c *Coordinator) SignInWorker(args *SignInWorkerArgs, reply *SignInWorkerReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	log.Printf("Coordinator: A worker signed in.\n")

	c.currentWorkersNum++
	return nil
}

func (c *Coordinator) PermitWorkerExit(args *PermitWorkerExitArgs, reply *PermitWorkerExitReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	reply.CanExit = len(c.doneSet) == c.nMap+c.nReduce
	return nil
}

func (c *Coordinator) SignOutWorker(args *SignOutWorkerArgs, reply *SignOutWorkerReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	log.Printf("Coordinator: A worker signed out.\n")

	c.currentWorkersNum--
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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return len(c.doneSet) == c.nMap+c.nReduce && c.currentWorkersNum == 0
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	// Initialize the coordinator.
	c := Coordinator{
		mapTasks:          []MapTask{},
		reduceTasks:       []ReduceTask{},
		readySet:          map[int]bool{},
		processingSet:     map[int]bool{},
		doneSet:           map[int]bool{},
		currentWorkersNum: 0,
		nMap:              len(files),
		nReduce:           nReduce,
		mutex:             sync.Mutex{},
	}

	// Allocate tasks.
	for i, file := range files {
		c.mapTasks = append(c.mapTasks, MapTask{
			MapTaskIndex: i,
			Filename:     file,
		})
	}
	for i := 0; i < nReduce; i++ {
		c.reduceTasks = append(c.reduceTasks, ReduceTask{
			ReduceTaskIndex: i,
		})
	}

	// Allocate ready set.
	for i := 0; i < c.nMap; i++ {
		c.readySet[i] = true
	}
	for i := 0; i < nReduce; i++ {
		c.readySet[i+c.nMap] = true
	}

	log.Println("Coordinator initialized.")

	// Start the RPC server.
	c.server()
	return &c
}

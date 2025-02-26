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

type TaskStatus int

const (
	InComplete TaskStatus = iota
	Complete
	Processing
)

type Task struct {
	ID     int
	Type   TaskType
	Status TaskStatus

	// For MapTask.
	MapFileName string
}

type Coordinator struct {
	tasks            []Task
	currentWorkerNum int
	nMap             int
	nReduce          int
	mutex            sync.Mutex
}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

func (c *Coordinator) RequestTask(args *RequestTaskArgs, reply *RequestTaskReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	allTasksCompleted := true
	allMapTasksCompleted := true
	for _, task := range c.tasks {
		if !allTasksCompleted && !allMapTasksCompleted {
			break
		}

		if task.Status != Complete {
			allTasksCompleted = false
			if task.Type == MapTask {
				allMapTasksCompleted = false
			}
		}
	}

	if allTasksCompleted {
		reply.Status = Completed
		return nil
	}

	if allMapTasksCompleted {
		// Allocate reduce task.
		for i := c.nMap; i < c.nMap+c.nReduce; i++ {
			if c.tasks[i].Status == InComplete {
				c.tasks[i].Status = Processing
				c.currentWorkerNum++

				reply.Status = Ready
				reply.Type = ReduceTask
				reply.ID = c.tasks[i].ID
				reply.NMap = c.nMap
				reply.NReduce = c.nReduce

				// log.Printf("Reduce task %d allocated\n", i)

				go c.checkTimeout(i)
				break
			}
		}
	} else {
		// Allocate map task firstly.
		for i := 0; i < c.nMap; i++ {
			if c.tasks[i].Status == InComplete {
				c.tasks[i].Status = Processing
				c.currentWorkerNum++

				reply.Status = Ready
				reply.Type = MapTask
				reply.ID = c.tasks[i].ID
				reply.NMap = c.nMap
				reply.NReduce = c.nReduce
				reply.MapFileName = c.tasks[i].MapFileName

				// log.Printf("Map task %d allocated\n", i)

				go c.checkTimeout(i)
				break
			}
		}
	}
	return nil
}

func (c *Coordinator) ReportTask(args *ReportTaskArgs, reply *ReportTaskReply) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if args.Type == MapTask {
		c.tasks[args.ID].Status = Complete
		// log.Printf("Map task %d completed\n", args.ID)
	} else {
		c.tasks[args.ID+c.nMap].Status = Complete
		// log.Printf("Reduce task %d completed\n", args.ID)
	}
	c.currentWorkerNum--
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

// Check timeout for a task.
func (c *Coordinator) checkTimeout(taskID int) {
	time.Sleep(10 * time.Second)
	c.mutex.Lock()
	defer c.mutex.Unlock()

	task := &c.tasks[taskID]
	if task.Status == Processing {
		// Timeout, reset the task to incomplete.
		task.Status = InComplete
		c.currentWorkerNum--
		/* if task.Type == MapTask {
			log.Printf("Map task %d timeout, reset to incomplete\n", taskID)
		} else {
			log.Printf("Reduce task %d timeout, reset to incomplete\n", taskID)
		} */
	}
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.currentWorkerNum > 0 {
		return false
	}
	for _, task := range c.tasks {
		if task.Status != Complete {
			return false
		}
	}
	return true
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		tasks:            make([]Task, len(files)+nReduce),
		currentWorkerNum: 0,
		nMap:             len(files),
		nReduce:          nReduce,
		mutex:            sync.Mutex{},
	}

	// Initialize map and reduce tasks.
	for i, file := range files {
		c.tasks[i] = Task{
			ID:          i,
			Type:        MapTask,
			Status:      InComplete,
			MapFileName: file,
		}
	}
	for i := c.nMap; i < c.nMap+c.nReduce; i++ {
		c.tasks[i] = Task{
			ID:     i - c.nMap,
			Type:   ReduceTask,
			Status: InComplete,
		}
	}

	c.server()
	return &c
}

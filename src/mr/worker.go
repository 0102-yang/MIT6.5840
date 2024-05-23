package mr

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"sort"
	"sync"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func calltaskDone(globalTaskNum int) {
	args := TaskDoneArgs{
		GlobalTaskNum: globalTaskNum,
	}
	reply := TaskDoneReply{}
	call("Coordinator.TaskDone", &args, &reply)
}

func processMapTask(task MapTaskReply, mapf func(string, string) []KeyValue) {
	// Open file.
	file, err := os.Open(task.MapFilename)
	if err != nil {
		log.Fatalf("Cannot open map file %v.", task.MapFilename)
	}
	defer file.Close()

	// Use buffered IO for reading.
	reader := bufio.NewReader(file)
	contents, err := io.ReadAll(reader)
	if err != nil {
		log.Fatalf("Cannot read content in %v.", task.MapFilename)
	}

	// Call map function to generate intermediate key-value pairs..
	keyValueArray := mapf(task.MapFilename, string(contents))

	// Partition key-value pairs into NReduce intermediate files.
	intermediate := make([][]KeyValue, task.NReduce)
	for _, kv := range keyValueArray {
		index := ihash(kv.Key) % task.NReduce
		intermediate[index] = append(intermediate[index], kv)
	}

	// Write NReduce intermediate key-value pairs bucket to seperate files.
	var wg sync.WaitGroup
	for reduceFileIndex, kva := range intermediate {
		wg.Add(1)
		go func(reduceFileIndex int, kva []KeyValue) {
			defer wg.Done()

			oname := fmt.Sprintf("mr-%v-%v", task.MapTaskIndex, reduceFileIndex)
			ofile, err := os.Create(oname)
			if err != nil {
				log.Fatalf("Cannot create file %v.", oname)
			}
			defer ofile.Close()

			// Use buffered IO for writing.
			writer := bufio.NewWriter(ofile)
			enc := json.NewEncoder(writer)
			for _, kv := range kva {
				enc.Encode(&kv)
			}
			writer.Flush()
		}(reduceFileIndex, kva)
	}

	// Wait for all goroutines to finish.
	wg.Wait()

	// Tell coordinator that this task is done.
	calltaskDone(task.MapTaskIndex)
}

func processReduceTask(task ReduceTaskReply, reducef func(string, []string) string) {
	// Do reduce task.
	intermediate := []KeyValue{}
	for mapFileIndex := 0; mapFileIndex < task.NMap; mapFileIndex++ {
		// Open intermediate file.
		oname := fmt.Sprintf("mr-%v-%v", mapFileIndex, task.ReduceTaskIndex)
		file, err := os.Open(oname)
		if err != nil {
			log.Fatalf("Cannot open %v", oname)
		}
		defer file.Close()

		// Use buffered IO for reading.
		reader := bufio.NewReader(file)
		dec := json.NewDecoder(reader)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
	}

	// Write to output file.
	oname := fmt.Sprintf("mr-out-%v", task.ReduceTaskIndex)
	ofile, err := os.Create(oname)
	if err != nil {
		log.Fatalf("Cannot create file %v", oname)
	}
	defer ofile.Close()

	sort.Slice(intermediate, func(i, j int) bool { return intermediate[i].Key < intermediate[j].Key })

	// Use buffered IO for writing.
	writer := bufio.NewWriter(ofile)
	for i := 0; i < len(intermediate); {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(writer, "%v %v\n", intermediate[i].Key, output)

		i = j
	}
	writer.Flush()

	// Tell coordinator that this task is done.
	// Get reduce num.
	args := GlobalTaskNumArgs{
		ReduceTaskNum: task.ReduceTaskIndex,
	}
	reply := GlobalTaskNumReply{}
	call("Coordinator.GetGlobalTaskNum", &args, &reply)
	calltaskDone(reply.GlobalTaskNum)
}

func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	// Sign in current worker to coordinator.
	signInArgs := SignInWorkerArgs{}
	signInReply := SignInWorkerReply{}
	call("Coordinator.SignInWorker", &signInArgs, &signInReply)

	// Loop to get map task and reduce task.
	for {
		// Get map task from coordinator.
		argsMap := MapTaskArgs{}
		replyMap := MapTaskReply{}
		call("Coordinator.GetMapTask", &argsMap, &replyMap)
		if replyMap.Valid {
			processMapTask(replyMap, mapf)
			continue
		}

		// Get reduce task from coordinator.
		argsReduce := ReduceTaskArgs{}
		replyReduce := ReduceTaskReply{}
		call("Coordinator.GetReduceTask", &argsReduce, &replyReduce)

		if replyReduce.ReduceTaskStatus == Ready {
			processReduceTask(replyReduce, reducef)
			continue
		}
		if replyReduce.ReduceTaskStatus == NotReady {
			time.Sleep(20 * time.Millisecond)
			continue
		}

		// Map tasks and reduce tasks are all done, just sign out and exit automatically.
		PermitWorkerExitArgs := PermitWorkerExitArgs{}
		PermitWorkerExitReply := PermitWorkerExitReply{}
		call("Coordinator.PermitWorkerExit", &PermitWorkerExitArgs, &PermitWorkerExitReply)

		if PermitWorkerExitReply.CanExit {
			signOutArgs := SignOutWorkerArgs{}
			signOutReply := SignOutWorkerReply{}
			call("Coordinator.SignOutWorker", &signOutArgs, &signOutReply)
			os.Exit(0)
		} else {
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatalf("Failed to connect to coordinator: %v", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err != nil {
		log.Fatalf("Failed to call function: %v due to %v.\n", rpcname, err.Error())
	}
}

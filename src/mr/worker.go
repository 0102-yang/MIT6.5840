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

func mapTask(mapTaskID int, mapFilename string, nReduce int, mapf func(string, string) []KeyValue) {
	// Open mapFile.
	mapFile, err := os.Open(mapFilename)
	if err != nil {
		log.Fatalf("Cannot open map file %v.", mapFilename)
	}
	defer mapFile.Close()

	// Use buffered IO for reading.
	reader := bufio.NewReader(mapFile)
	contents, err := io.ReadAll(reader)
	if err != nil {
		log.Fatalf("Cannot read content in %v.", mapFilename)
	}

	// Call map function to generate intermediate key-value pairs..
	tmp_kv := mapf(mapFilename, string(contents))

	// Partition key-value pairs into NReduce intermediate files.
	intermediate := make([][]KeyValue, nReduce)
	for _, kv := range tmp_kv {
		index := ihash(kv.Key) % nReduce
		intermediate[index] = append(intermediate[index], kv)
	}

	// Write NReduce intermediate key-value pairs bucket to seperate files.
	// Wait for all goroutines to finish.
	var wg sync.WaitGroup
	for intermediateFileIndex, kva := range intermediate {
		wg.Add(1)
		go func(reduceFileIndex int, kva []KeyValue) {
			defer wg.Done()

			intermediateFilename := fmt.Sprintf("mr-%v-%v", mapTaskID, reduceFileIndex)
			ofile, err := os.Create(intermediateFilename)
			if err != nil {
				log.Fatalf("Cannot create file %v.", intermediateFilename)
			}
			defer ofile.Close()

			// Use buffered IO for writing.
			writer := bufio.NewWriter(ofile)
			enc := json.NewEncoder(writer)
			for _, kv := range kva {
				enc.Encode(&kv)
			}
			writer.Flush()
		}(intermediateFileIndex, kva)
	}
	wg.Wait()
}

func reduceTask(reduceTaskID int, nMap int, reducef func(string, []string) string) {
	// Read all intermediate files.
	intermediateFilenames := []string{}
	for intermediateFileIndex := 0; intermediateFileIndex < nMap; intermediateFileIndex++ {
		intermediateFilename := fmt.Sprintf("mr-%v-%v", intermediateFileIndex, reduceTaskID)
		intermediateFilenames = append(intermediateFilenames, intermediateFilename)
	}

	intermediate := []KeyValue{}
	var intermediateMutex sync.Mutex
	var wg sync.WaitGroup
	for intermediateFileIndex, intermediateFilename := range intermediateFilenames {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()

			// Open intermediate file.
			intermediateFilename := fmt.Sprintf("mr-%v-%v", intermediateFileIndex, reduceTaskID)
			intermediateFile, err := os.Open(intermediateFilename)
			if err != nil {
				log.Fatalf("Cannot open %v", intermediateFilename)
			}
			defer intermediateFile.Close()

			// Use buffered IO for reading.
			reader := bufio.NewReader(intermediateFile)
			dec := json.NewDecoder(reader)
			for {
				var kv KeyValue
				if err := dec.Decode(&kv); err != nil {
					break
				}

				intermediateMutex.Lock()
				intermediate = append(intermediate, kv)
				intermediateMutex.Unlock()
			}
		}(intermediateFilename)
	}
	wg.Wait()

	// Write to output file.
	outputFilename := fmt.Sprintf("mr-out-%v", reduceTaskID)
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf("Cannot create file %v", outputFilename)
	}
	defer outputFile.Close()

	sort.Slice(intermediate, func(i, j int) bool { return intermediate[i].Key < intermediate[j].Key })

	// Use buffered IO for writing.
	writer := bufio.NewWriter(outputFile)
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
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	for {
		args := RequestTaskArgs{}
		reply := RequestTaskReply{}
		if !call("Coordinator.RequestTask", &args, &reply) {
			log.Println("Cannot request task from coordinator. Maybe coordinator is down. Exit.")
			os.Exit(0)
		}

		switch reply.Status {
		case NotReady:
			// Sleep for a while.
			time.Sleep(1 * time.Second)
		case Ready:
			switch reply.Type {
			case MapTask:
				mapTask(reply.ID, reply.MapFileName, reply.NReduce, mapf)
			case ReduceTask:
				reduceTask(reply.ID, reply.NMap, reducef)
			}

			reportArgs := ReportTaskArgs{
				ID:   reply.ID,
				Type: reply.Type,
			}
			reportReply := ReportTaskReply{}
			call("Coordinator.ReportTask", &reportArgs, &reportReply)
		case Completed:
			os.Exit(0)
		}
	}

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

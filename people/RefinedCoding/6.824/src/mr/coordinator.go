package mr

import (
	"encoding/base64"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

/*
cp people/RefinedCoding/6.824/src/mr/rpc.go homework/0528/RefinedCoding-rpc.go
cp people/RefinedCoding/6.824/src/mr/coordinator.go homework/0529/RefinedCoding-coordinator.go
cp people/RefinedCoding/6.824/src/mr/worker.go homework/0530/RefinedCoding-worker.go
*/

const TaskTimeOut = 10

type TaskType int
type TaskStatus int

// https://www.educative.io/edpresso/what-is-an-enum-in-golang
const (
	MapTask TaskType = iota
	ReduceTask
	None
	Exit
)

const (
	New TaskStatus = iota
	InProgress
	Done
)

type Task struct {
	Id int
	Type TaskType
	File string
	WorkerId int
	Status TaskStatus
}

type Coordinator struct {
	// Your definitions here.
	mutex sync.Mutex
	mapTasks []Task
	reduceTasks []Task
}

func (c *Coordinator) SelectTask(workerId int) *Task {
	var tasks []Task
	if len(c.mapTasks) > 0 {
		tasks = c.mapTasks
	} else {
		tasks = c.reduceTasks
	}
	for _, task := range tasks {
		if task.Status == New {
			task.Status = InProgress
			return &task
		}
	}
	return &Task { -1, None, "", -1, Done }
}

func (c *Coordinator) CheckTaskStatus(task *Task) {
	if task.Type == MapTask || task.Type == ReduceTask {
		<- time.After(time.Second * TaskTimeOut)
		c.mutex.Lock()
		defer c.mutex.Unlock()
		if task.Status == InProgress {
			task.Status = New
			task.WorkerId = -1
		}
	}
}

// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

func (c *Coordinator) TaskRequestHandler(request *TaskRequest, response *TaskResponse) error {
	c.mutex.Lock()
	var task Task = c.SelectTask()
	task.WorkerId = request.WorkerId;

	response.TaskId = task.Id
	response.TaskType = task.Type
	response.TaskFile = task.File

	c.mutex.Unlock()
	go c.CheckTaskStatus(task)

	return nil
}

func (c *Coordinator) TaskStatusHandler(request *TaskStatusRequest, response *TaskStatusResponse) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// TODO: var task *Task = c.Tasks[request.TaskId]
	var task *Task
	if  request.TaskType == MapTask {
		task = &c.mapTasks[request.TaskId]
	} else if request.TaskType == ReduceTask {
		task = &c.reduceTasks[request.TaskId]
	} else {
		// TODO: check error
	}

	if task.WorkerId == request.WorkerId && task.Status == InProgress {
		task.Status = Done
		// TODO: c.Tasks[request.TaskType]
		if request.TaskType == MapTask {
			c.mapTasks = append(c.mapTasks[:request.TaskId], c.mapTasks[request.TaskId + 1 : ]...)
		} else if request.TaskType == ReduceTask {
			c.reduceTasks = append(c.reduceTasks[:request.TaskId], c.reduceTasks[request.TaskId + 1 : ]...)
		}
	}

	// TODO: update task status and return all task done ?
	response.Done = len(c.mapTasks) == 0 && len(c.reduceTasks) == 0

	return nil
}

func (c *Coordinator) ReduceCountHandler(request *ReduceCountRequest, response *ReduceCountResponse) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	response.ReduceCount = len(c.reduceTasks)

	// return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	socket := coordinatorSock()
	os.Remove(socket)
	l, e := net.Listen("unix", socket)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {

	// Your code here.
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return len(c.mapTasks) == 0 && len(c.reduceTasks) == 0
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	c.mapTasks = make([]Task, 0, len(files))
	c.reduceTasks = make([]Task, 0, nReduce)

	for i := 0; i < len(files); i ++ {
		c.mapTasks = append(c.mapTasks, Task { i, MapTask, files[i], -1, New })
	}

	for i := 0; i < nReduce; i ++ {
		c.reduceTasks = append(c.reduceTasks, Task { i, ReduceTask, nil, -1, New })
	}

	c.server()

	outputs, _ = filepath.Glob("mr-out-*")
	for _, f := range outputs {
		if err := os.Remove(f); err != nil {
			fmt.Errorf("Failed to remove file %v\n", f)
		}
	}

	// TODO: create folder for intermediate results

	return &c
}

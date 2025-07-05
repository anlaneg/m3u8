package tool

import (
	"fmt"
	"sync"
	//"time"
)

type ConcurrencyRun interface {
	GetConcurrency() int
	//SetTasks([]interface{})
	//IsFinished() bool
	//GetNextTask() (interface{}, error)
	DoTask(interface{}) error
}

type ConcurrencyTask struct {
	concurrency int
	tasks       []interface{}
}

/*func (c *ConcurrencyTask) GetConcurrency() int {
	return c.concurrency
}

func (c *ConcurrencyTask) SetTasks(tasks []interface{}) {
	c.tasks = tasks
}*/

func (c *ConcurrencyTask) IsFinish() bool {
	if c.tasks == nil {
		return true
	}

	return len(c.tasks) == 0
}

func (c *ConcurrencyTask) GetNextTask() (interface{}, error) {
	if c.IsFinish() {
		return nil, fmt.Errorf("task finished")
	}

	task := c.tasks[0]
	c.tasks = c.tasks[1:]
	return task, nil
}

/*func (c *ConcurrencyTask) DoTask(interface{}) error {
	return fmt.Errorf("not implement")
}*/

func ConcurrencyTaskRun(run ConcurrencyRun, tasks []interface{}) {
	c := &ConcurrencyTask{concurrency: run.GetConcurrency(), tasks: tasks}
	//c.SetTasks(tasks)

	wg := sync.WaitGroup{}
	limit := c.concurrency
	if limit <= 0 {
		limit = 20
	}

	limitChan := make(chan int, limit)
	for !c.IsFinish() {
		task, _ := c.GetNextTask()
		wg.Add(1)
		go func() {
			defer func() {
				<-limitChan
				wg.Done()
			}()
			if err := run.DoTask(task); err != nil {
				fmt.Printf("c.DoTask failed %s\n", err.Error())
			}
		}()
		limitChan <- 1
	}
	wg.Wait()
}

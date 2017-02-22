package main

import "regexp"

// Task is a simple interface for potentially concurrent tasks
type Task interface {
	Start() <-chan string
	SetConcurrency(uint8)
}

type cfPipeline struct {
	tasks      <-chan []Task
	filter     *regexp.Regexp
	cLimit     uint8
	cLimitSem  chan struct{}
	cLimitLock chan struct{}
	results    chan string
}

func (cfp *cfPipeline) execute(task Task, wg chan struct{}) {
	<-cfp.cLimitSem

	results := task.Start()
	for res := range results {
		if cfp.filter.MatchString(res) {
			cfp.results <- res
		}
	}

	wg <- struct{}{}
	cfp.cLimitSem <- struct{}{}
}

func (cfp *cfPipeline) Start() <-chan string {
	go func() {
		for taskBundle := range cfp.tasks {
			wg := make(chan struct{})
			for _, task := range taskBundle {
				go cfp.execute(task, wg)
			}
			for i := 0; i < len(taskBundle); i++ {
				<-wg
			}
		}
		close(cfp.results)
	}()

	return cfp.results
}

func (cfp *cfPipeline) SetConcurrency(newCLimit uint8) {
	if newCLimit < 1 {
		panic("such a concurrency limit is not allowed")
	}

	cfp.cLimitLock <- struct{}{}
	defer func() {
		<-cfp.cLimitLock
	}()

	if newCLimit > cfp.cLimit {
		for i := uint8(0); i < newCLimit-cfp.cLimit; i++ {
			cfp.cLimitSem <- struct{}{}
		}
	} else {
		for i := uint8(0); i < cfp.cLimit-newCLimit; i++ {
			<-cfp.cLimitSem
		}
	}
	cfp.cLimit = newCLimit
}

// NewFilterPipeline creates and initializes the new task
func NewFilterPipeline(tasks <-chan []Task, concurrencyLimit uint8, filter *regexp.Regexp) Task {
	result := &cfPipeline{
		tasks:      tasks,
		filter:     filter,
		cLimitSem:  make(chan struct{}, 256),
		cLimitLock: make(chan struct{}, 1),
		results:    make(chan string),
	}
	result.SetConcurrency(concurrencyLimit)

	return result
}

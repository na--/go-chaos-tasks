package main

import (
	"regexp"
	"testing"
	"time"
)

// tt stands for test task
type tt []struct {
	sleep  time.Duration
	result string
}

func (t tt) SetConcurrency(count uint8) { /* noop, not used in tests */ }
func (t tt) Start() <-chan string {
	results := make(chan string)
	go func() {
		for _, preparedResult := range t {
			time.Sleep(preparedResult.sleep * time.Millisecond)
			results <- preparedResult.result
		}
		close(results)
	}()

	return results
}

func TestWithNoTasks(t *testing.T) {
	t.Parallel()

	tasks := make(chan []Task)
	fp := NewFilterPipeline(tasks, 1, regexp.MustCompile("^fizz.*"))
	results := fp.Start()
	close(tasks)

	if res, ok := <-results; ok {
		t.Errorf("Received a result '%s' when not expecting anything!", res)
	}
}

func TestWithOneTasks(t *testing.T) {
	t.Parallel()

	tasks := make(chan []Task)
	fp := NewFilterPipeline(tasks, 3, regexp.MustCompile("^fizz.*"))
	fp.SetConcurrency(1)
	results := fp.Start()

	expResult := "fizzy"
	tasks <- []Task{tt{{3, expResult}}}
	close(tasks)

	if res, ok := <-results; !ok {
		t.Errorf("Did not receive the expected result")
	} else if res != expResult {
		t.Errorf("Received a result '%s' when expecting '%s'!", res, expResult)
	}

	if res, ok := <-results; ok {
		t.Errorf("Received a result '%s' when not expecting anything!", res)
	}
}

func TestWithMoreTasks(t *testing.T) {
	t.Parallel()

	tasks := make(chan []Task)
	fp := NewFilterPipeline(tasks, 3, regexp.MustCompile(".*buzz$"))
	fp.SetConcurrency(1)
	results := fp.Start()
	fp.SetConcurrency(2)

	go func() {
		tasks <- []Task{
			tt{{1, "wat"}, {30, "fizzybuzz"}, {500, "buzz"}, {1, "watwat"}},
			tt{{11, "ba"}, {11, "dum"}, {17, "tsss"}},
			tt{{150, "fizzbuzz"}, {20, "brumbuzz"}},
		}
		fp.SetConcurrency(3)
		tasks <- []Task{
			tt{{15, "a buzz"}},
			tt{{3, "fizz"}, {183, "fizzy"}},
		}
		tasks <- []Task{
			tt{{13, "NaNaNaNaNaNaNaNaNa"}, {37, "BATMAN!"}},
			tt{{5, "an aldrin buzz"}, {25, "another buzz"}},
		}
		fp.SetConcurrency(1)
		tasks <- []Task{
			tt{{1, "bye"}, {1, "bye"}},
			tt{{5, "the final buzz"}},
		}
		close(tasks)
	}()

	expResults := []string{
		"fizzybuzz", "fizzbuzz", "brumbuzz", "buzz", "a buzz",
		"an aldrin buzz", "another buzz", "the final buzz",
	}

	for i, expResult := range expResults {
		if res, ok := <-results; !ok {
			t.Errorf("The result channel was closed prematurely when expecting valid result #%d: %s", i, expResult)
		} else if res != expResult {
			t.Errorf("Received a result '%s' (#%d) when expecting '%s'!", res, i, expResult)
		}
	}

	if res, ok := <-results; ok {
		t.Errorf("Received a result '%s' when not expecting anything!", res)
	}
}

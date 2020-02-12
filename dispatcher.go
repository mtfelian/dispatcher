package dispatcher

import (
	"sync"
	"time"

	"github.com/mtfelian/synced"
)

type (
	// Treator can treat something
	Treator interface {
		Treat() (interface{}, error)
	}

	// Result consists of an input and output data
	Result struct {
		In    interface{}
		Out   interface{}
		Error error
	}

	// OnResultFunc is a func to do something with task result on it's finish
	OnResultFunc func(result Result)

	// Dispatcher implements a task-dispatching functionality
	Dispatcher struct {
		taskQueue synced.Queue

		workers    synced.Counter
		maxWorkers int
		tasksDone  synced.Counter
		errorCount synced.Counter

		inputC  chan Treator
		resultC chan Result
		stopC   chan struct{}

		stopping      bool
		stoppingMutex sync.Mutex

		onResultFunc OnResultFunc
	}
)

// New returns new dispatcher with max workers
func New(max int, onResultFunc OnResultFunc) *Dispatcher {
	return &Dispatcher{
		taskQueue:    synced.NewQueue(),
		workers:      synced.NewCounter(0),
		maxWorkers:   max,
		tasksDone:    synced.NewCounter(0),
		errorCount:   synced.NewCounter(0),
		inputC:       make(chan Treator),
		resultC:      make(chan Result),
		stopC:        make(chan struct{}),
		onResultFunc: onResultFunc,
	}
}

// setStopping sets state to stopping
func (d *Dispatcher) setStopping() {
	d.stoppingMutex.Lock()
	d.stopping = true
	d.stoppingMutex.Unlock()
}

// isStopping returns whether the dispatcher is stopping
func (d *Dispatcher) isStopping() bool {
	d.stoppingMutex.Lock()
	defer d.stoppingMutex.Unlock()
	return d.stopping
}

// Workers returns current worker count
func (d *Dispatcher) Workers() int { return d.workers.Get() }

// SetMaxWorkers to n
func (d *Dispatcher) SetMaxWorkers(n int) { d.maxWorkers = n }

// ErrorCount returns an error count
func (d *Dispatcher) ErrorCount() int { return d.errorCount.Get() }

// Stop the dispatcher
func (d *Dispatcher) Stop() {
	if d.isStopping() {
		return
	}
	d.setStopping()
	d.stopC <- struct{}{}
	close(d.stopC)
}

// AddWork adds work with data to the dispatcher
func (d *Dispatcher) AddWork(data Treator) {
	if !d.isStopping() {
		d.inputC <- data
	}
}

// FillWork adds work with data to the dispatcher after queue size will fall below queueLimit value,
// check it every checkInterval
func (d *Dispatcher) FillWork(data Treator, queueLimit int, checkInterval time.Duration) {
	for d.QueueLen() > queueLimit { // to prevent queue major growth
		time.Sleep(checkInterval)
		continue
	}
	go d.AddWork(data)
}

// TasksDone returns done tasks count
func (d *Dispatcher) TasksDone() int { return d.tasksDone.Get() }

// WaitUntilNoTasks stops the dispatcher making checks for "0 tasks now" every period
func (d *Dispatcher) WaitUntilNoTasks(period time.Duration) {
	for {
		if d.Workers() == 0 {
			break
		}
		time.Sleep(period)
	}
	d.Stop()
}

// treat the element
func (d *Dispatcher) treat(element Treator) {
	result, err := element.Treat()
	d.sendResult(Result{In: element, Out: result, Error: err})
}

// sendResult to results channel
func (d *Dispatcher) sendResult(result Result) {
	if !d.isStopping() {
		d.resultC <- result
	}
}

// popTreat pops an element from queue q and treats it
func (d *Dispatcher) popTreat(q *synced.Queue) {
	popped, err := q.Pop()
	if err != nil {
		d.sendResult(Result{In: nil, Out: nil, Error: err})
		return
	}
	go d.treat(popped.(Treator))
}

// QueueLen returns current length of the queue
func (d *Dispatcher) QueueLen() int { return d.taskQueue.Len() }

// Run starts dispatching
func (d *Dispatcher) Run() {
	for {
		select {
		case r := <-d.resultC: // worker is done
			if r.Error != nil {
				d.errorCount.Inc()
			}
			go d.onResultFunc(r)

			d.tasksDone.Inc()
			d.workers.Dec()
			// queue is not empty -- can add worker
			if d.taskQueue.Len() > 0 {
				d.workers.Inc()
				d.popTreat(&d.taskQueue)
			}
		case data := <-d.inputC: // dispatcher receives some data
			if d.workers.Get() >= d.maxWorkers {
				d.taskQueue.Push(data)
				continue
			}
			d.workers.Inc()

			// queue is not empty, FIFO - push, pop
			if d.taskQueue.Len() > 0 {
				d.taskQueue.Push(data)
				d.popTreat(&d.taskQueue)
				continue
			}

			// queue is empty, simply treat the url
			go d.treat(data)
		case <-d.stopC: // stop signal received
			// cleaning up, may be
			t := time.NewTicker(100 * time.Millisecond)
			for {
				select {
				case <-d.resultC:
				case <-d.inputC:
				case <-t.C:
					if d.Workers() == 0 {
						t.Stop()
						return
					}
				}
			}
		}
	}
}

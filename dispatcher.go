package dispatcher

import (
	"sync"

	"github.com/mtfelian/dispatcher/counter"
	"github.com/mtfelian/dispatcher/queue"
)

type (
	// Result is a testOutput data consisting of input and output data
	Result struct {
		In  interface{}
		Out interface{}
	}

	// channels used by dispatcher
	channels struct {
		// input is an input input
		input chan interface{}
		// leave <- send here when worker stops
		leave chan error
		// results <- here at worker ends
		results chan Result
		// stop <- send here to stop dispatcher
		stop chan bool
	}

	// TreatFunc is a func for treating generic input element
	TreatFunc func(element interface{}) (interface{}, error)

	// OnResultFunc is a func to perform with testOutput on task end
	OnResultFunc func(result Result)
)

// dispatcher
type Dispatcher struct {
	sync.Mutex
	workers      counter.Synced
	maxWorkers   int
	tasksDone    counter.Synced
	errorCount   counter.Synced
	signals      channels
	treatFunc    TreatFunc
	onResultFunc OnResultFunc
}

// New returns new dispatcher with max workers
func New(max int, treatFunc TreatFunc, onResultFunc OnResultFunc) *Dispatcher {
	return &Dispatcher{
		workers:    counter.New(0),
		maxWorkers: max,
		tasksDone:  counter.New(0),
		errorCount: counter.New(0),
		signals: channels{
			input:   make(chan interface{}),
			leave:   make(chan error),
			results: make(chan Result),
			stop:    make(chan bool),
		},
		treatFunc:    treatFunc,
		onResultFunc: onResultFunc,
	}
}

// _signals returns signals struct thread-safely
func (d *Dispatcher) _signals() *channels {
	d.Lock()
	defer d.Unlock()
	return &d.signals
}

// Workers returns current worker count
func (d *Dispatcher) Workers() int { return d.workers.Get() }

// SetMaxWorkers to n
func (d *Dispatcher) SetMaxWorkers(n int) { d.maxWorkers = n }

// Stop the dispatcher
func (d *Dispatcher) Stop() { d._signals().stop <- true }

// AddWork adds work input to dispatcher
func (d *Dispatcher) AddWork(data interface{}) { d._signals().input <- data }

// TasksDone returns done tasks count
func (d *Dispatcher) TasksDone() int { return d.tasksDone.Get() }

// treat the element
func (d *Dispatcher) treat(element interface{}) {
	result, err := d.treatFunc(element)
	if err != nil {
		d._signals().leave <- err
		return
	}
	d._signals().results <- Result{In: element, Out: result}
	d._signals().leave <- nil
}

// popTreat pops an element from queue q and treats it
func (d *Dispatcher) popTreat(q *queue.Synced) {
	popped, err := q.Pop()
	if err != nil {
		d._signals().leave <- err
		return
	}
	go d.treat(popped.(interface{}))
}

// Run starts dispatching
func (d *Dispatcher) Run() {
	q := queue.New()
	for {
		select {
		case <-d._signals().leave: // worker is done
			d.tasksDone.Inc()
			d.workers.Dec()
			// queue is not empty -- can add worker
			if q.Len() > 0 {
				d.workers.Inc()
				d.popTreat(&q)
			}
		case data := <-d._signals().input: // dispatcher receives input
			if d.workers.Get() >= d.maxWorkers {
				q.Push(data)
				continue
			}
			d.workers.Inc()

			// queue is not empty, FIFO - push, pop
			if q.Len() > 0 {
				q.Push(data)
				d.popTreat(&q)
				continue
			}

			// queue is empty, simply treat the url
			go d.treat(data)
		case result := <-d._signals().results: // received a testOutput from worker
			d.onResultFunc(result)
		case <-d._signals().stop: // stop signal received
			return
		}
	}
}

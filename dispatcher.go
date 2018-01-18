package dispatcher

import (
	"github.com/mtfelian/dispatcher/counter"
	"github.com/mtfelian/dispatcher/queue"
)

type (
	// Gi is a generic input type
	Gi interface{}
	// Gr is a generic resulting type
	Gr interface{}
	// Result is a testOutput data consisting of Gi and Gr to easily link the input and output data
	Result struct {
		in  Gi
		out Gr
	}

	// Channels used by dispatcher
	Channels struct {
		// data is an input data
		data chan Gi
		// leave <- send here when worker stops
		leave chan error
		// results <- here at worker ends
		results chan Result
		// stop <- send here to stop dispatcher
		stop chan bool
	}

	// TreatFunc is a func for treating generic input element
	TreatFunc func(element Gi) (Gr, error)

	// OnResultFunc is a func to perform with testOutput on task end
	OnResultFunc func(result Result)
)

// InitChannels initializes channels for dispatcher
func InitChannels() Channels {
	return Channels{
		data:    make(chan Gi),
		leave:   make(chan error),
		results: make(chan Result),
		stop:    make(chan bool),
	}
}

// dispatcher
type Dispatcher struct {
	workers      counter.Synced
	maxWorkers   int
	tasksDone    counter.Synced
	errorCount   counter.Synced
	signals      Channels
	treatFunc    TreatFunc
	onResultFunc OnResultFunc
}

// NewDispatcher returns new dispatcher with max workers
func NewDispatcher(signals Channels, max int, treatFunc TreatFunc, onResultFunc OnResultFunc) *Dispatcher {
	return &Dispatcher{
		workers:      counter.New(0),
		maxWorkers:   max,
		tasksDone:    counter.New(0),
		errorCount:   counter.New(0),
		signals:      signals,
		treatFunc:    treatFunc,
		onResultFunc: onResultFunc,
	}
}

// Workers returns current worker count
func (d *Dispatcher) Workers() int { return d.workers.Get() }

// SetMaxWorkers to n
func (d *Dispatcher) SetMaxWorkers(n int) { d.maxWorkers = n }

// Stop the dispatcher
func (d *Dispatcher) Stop() { d.signals.stop <- true }

// TasksDone returns done tasks count
func (d *Dispatcher) TasksDone() int { return d.tasksDone.Get() }

// treat the element
func (d *Dispatcher) treat(element Gi) {
	result, err := d.treatFunc(element)
	if err != nil {
		d.signals.leave <- err
		return
	}
	d.signals.results <- Result{in: element, out: result}
	d.signals.leave <- nil
}

// popTreat pops an element from queue q and treats it
func (d *Dispatcher) popTreat(q *queue.Synced) {
	popped, err := q.Pop()
	if err != nil {
		d.signals.leave <- err
		return
	}
	go d.treat(popped.(Gi))
}

// Run starts dispatching
func (d *Dispatcher) Run() {
	q := queue.New()
	for {
		select {
		case <-d.signals.leave: // worker is done
			d.tasksDone.Inc()
			d.workers.Dec()
			// queue is not empty -- can add worker
			if q.Len() > 0 {
				d.workers.Inc()
				d.popTreat(&q)
			}
		case data := <-d.signals.data: // dispatcher receives data
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
		case result := <-d.signals.results: // received a testOutput from worker
			d.onResultFunc(result)
		case <-d.signals.stop: // stop signal received
			return
		}
	}
}

[![Go Report Card](https://goreportcard.com/badge/github.com/mtfelian/dispatcher)](https://goreportcard.com/report/github.com/mtfelian/dispatcher)
[![GoDoc](https://godoc.org/github.com/mtfelian/dispatcher?status.png)](http://godoc.org/github.com/mtfelian/dispatcher)
[![Build status](https://travis-ci.org/mtfelian/dispatcher.svg?branch=master)](https://travis-ci.org/mtfelian/dispatcher)

# Dispatcher

Dispatcher provides a concurrent tasks treating functionality based
on go-routines.

After you initialize and run dispatcher it is ready to perform tasks.

You can add tasks via `AddWork()` func. Both `Run()` and `AddWork()`
must be called in separate go-routines (please see example below).
Tasks will be treated concurrently but amount of workers will be limited
to the parameter specified at the dispatcher initialization
call `New()`.

To stop the dispatcher you can use the `Stop()` func. But you should
be careful with it: just after this call the dispatcher stops
receiving new work and stops returning results even if the task was
already processed but result were not yet sent into appropriate channel.
When `Stop()` was called, dispatcher flushes channels and exits.

If you need just to do tons of work with dispatcher and quit after
without waiting for next portions of work, you can use the
`WaitUntilNoTasks(period)` func. It checks the number of currently
processing tasks each period of seconds. If it seems that dispatcher
is not processing any task, it stops.

The `onResultFunc` is thread-safe so don't do anything time-consuming
in it. If you need to perform some time-consuming operations in 
`onResultFunc` then you should use the second dispatcher and organize 
two dispatchers in a conveyor pattern using `treatFunc` in first 
dispatcher to treat first stage data, and in the `onResultFunc` of the 
first dispatcher you may call `go AddWork(...)` to send the result to 
the second dispatcher. The second dispatcher will treat the data from 
the first's `onResultFunc` in it's own `treatFunc` and return it in 
it's own `onResultFunc`, this one of a second stage dispatcher. 
And so on: you can organize it in more stages.

Any one dispatcher performs only one `onResultFunc` at a time.

The advantage of a thread-safe `onResultFunc` is that you can use in it
external variables like package variables, func scope or any other
shared memory, for example, append to slice or change map key.

The `treatFunc` is not thread-safe and one dispatcher performs
multiple of `treatFunc` at a time. Be careful using shared memory in it
to avoid data races.

Here is one-staged usage example:

```
import (
    ...
    "github.com/mtfelian/dispatcher"
    ...
)
...

// prepareDispatcher via parser p, fResults is an output file for parsed results,
// fParsed is an output file to store parsed locations, skip is a slice of locations to skip (already parsed before).
// Limits concurrent workers to the given amount.
func prepareDispatcher(p parser.Parser, fResults, fParsed *os.File, skip []string, workers int) *dispatcher.Dispatcher {
	// treatFunc parses a location stored in element, returns a parsed record
	treatFunc := func(element interface{}) (interface{}, error) {
		location := element.(string)
		if SliceContains(location, skip) { // this URL already failed
			log.Printf("INFO >> %s already parsed\n", location)
			return nil, nil
		}
		record, err := p.Scrape(location)
		if record != nil {
			bookRecord := record.(*Record)
			if err != nil || bookRecord.ID == 0 { // failed to parse
				return nil, fmt.Errorf("failed to parse at location %s", location)
			}
		}
		return record, nil
	}

	// onResultFunc logs output data and stores the result
	onResultFunc := func(result dispatcher.Result) {
		if result.Error != nil {
			log.Printf("ERROR %s", result.Error.Error())
			return
		}
		if result.Out == nil { // && result.Error == nil, skipped without error
			return
		}
		// parse successful
		location := result.In.(string)
		if _, err := fParsed.WriteString(location + "\n"); err != nil {
			log.Printf("ERROR writing parsed URL to file %s: %v\n", location, err)
		}

		jsonData, err := json.Marshal(result.Out.(*Record))
		if err != nil {
			log.Printf("ERROR failed to marshal JSON %s: %v\n", location, err)
		}
		if _, err := fResults.WriteString(string(jsonData) + "\n"); err != nil {
			log.Printf("ERROR writing JSON data to file %s: %v\n", location, err)
		}
	}
	return dispatcher.New(workers, treatFunc, onResultFunc)
}

// Parse via given parser
func Parse(p parser.Parser) {
	...
	d := prepareDispatcher(p, fResults, fParsed, skip, 10)
	go d.Run()
	log.Println("INFO started dispatcher")
	for _, url := range urls {
		go d.AddWork(url)
	}

	// check for no tasks every 5 second
	d.WaitUntilNoTasks(5 * time.Second)
	...
}
```

Also you can explore the tests.

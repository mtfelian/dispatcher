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

The `Treat()` by default is not thread-safe and one dispatcher performs
multiple of `Treat` at a time. Be careful using shared memory in it
to avoid data races. The `onResultFunc` also is not thread-safe.
You should control safety by yourself.

You can use the second dispatcher and organize two dispatchers in
a conveyor pattern using `Treat` in first dispatcher to treat first
stage data, and in the `onResultFunc` of the first dispatcher you may
call `go AddWork(...)` to send the result to the second dispatcher.
The second dispatcher will treat the data from the first's
`onResultFunc` in it's own `Treat` of the returned element and return it
in it's own  `onResultFunc`, this one of a second stage dispatcher.
And so on: you can organize it in more stages.

For example of one-staged using please look into the tests.

[![Go Report Card](https://goreportcard.com/badge/github.com/mtfelian/dispatcher)](https://goreportcard.com/report/github.com/mtfelian/dispatcher)

[![GoDoc](https://godoc.org/github.com/mtfelian/dispatcher?status.png)](http://godoc.org/github.com/mtfelian/dispatcher)

# Dispatcher

Dispatcher provides a tasks concurrent treating functionality based
on go-routines.

Here is an usage example:

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
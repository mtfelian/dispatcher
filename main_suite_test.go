/*
In this example the test application launches the HTTP server.
This HTTP server handles route POST /treat
Also it launches mock HTTP server which used out of API request to POST /treat via treatFunc : http.Get(in.URL)
I.e. data sent to the dispatcher are treated via treatFunc call.
Then result is transmitted to onResultFunc of the dispatcher and you will see it
on the console stdout as "Count for..."
Launch with 'ginkgo -race' or 'ginkgo -race -untilItFails' for infinite testing.
*/
package dispatcher_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	d "github.com/mtfelian/dispatcher"
	"github.com/mtfelian/dispatcher/counter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	mServer    *mockServer
	dispatcher *d.Dispatcher
	server     *httptest.Server

	totalFound = counter.New(0)
	maxWorkers = 9
)

// testInput is an input for apiTreat method
type testInput struct {
	URL  string `json:"url"`
	What string `json:"what"`
}

// testOutput is an output from a dispatcher
type testOutput struct {
	n   int
	url string
}

// testHandler is a test HTTP handler
type testHandler struct{}

// ServeHTTP is a test HTTP server handler
func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		return
	}

	var input testInput
	if err := json.Unmarshal(body, &input); err != nil {
		fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
		return
	}

	go func() { dispatcher.AddWork(input) }()
	w.WriteHeader(http.StatusOK)
}

// TestApp runs all test suites
func TestApp(t *testing.T) {
	server = httptest.NewServer(&testHandler{})
	defer server.Close()

	// init mock server
	mServer = newMockServer()
	defer mServer.Close()

	treatFunc := func(element interface{}) (interface{}, error) {
		in := element.(testInput)
		response, err := http.Get(in.URL)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		respBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		return testOutput{n: strings.Count(string(respBytes), in.What), url: in.URL}, nil
	}

	onResultFunc := func(r d.Result) {
		n := r.Out.(testOutput).n
		fmt.Printf("Count for %s: %d\n", r.In.(testInput).URL, n)
		totalFound.Add(n)
	}

	dispatcher = d.NewDispatcher(maxWorkers, treatFunc, onResultFunc)
	go dispatcher.Run()

	rand.Seed(time.Now().UnixNano())

	RegisterFailHandler(Fail)
	RunSpecs(t, "Dispatcher Suite")
}

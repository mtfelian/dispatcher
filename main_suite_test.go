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
	"github.com/mtfelian/synced"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	mServer    *mockServer
	dispatcher *d.Dispatcher
	server     *httptest.Server

	totalFound = synced.NewCounter(0)
	maxWorkers = 9
)

// testInput is an input for apiTreat method
type testInput struct {
	URL  string `json:"url"`
	What string `json:"what"`
}

// Treat defines how the test input should be treated
func (e testInput) Treat() (interface{}, error) {
	response, err := http.Get(e.URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	respBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return testOutput{n: strings.Count(string(respBytes), e.What), url: e.URL}, nil
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

	onResultFunc := func(r d.Result) {
		if r.Error == nil {
			n := r.Out.(testOutput).n
			fmt.Printf("Count for %s: %d\n", r.In.(testInput).URL, n)
			totalFound.Add(n)
			return
		}
		fmt.Printf("Error for %s: %v\n", r.In.(testInput), r.Error)
	}

	dispatcher = d.New(maxWorkers, onResultFunc)
	go dispatcher.Run()

	rand.Seed(time.Now().UnixNano())

	RegisterFailHandler(Fail)
	RunSpecs(t, "Dispatcher Suite")
}

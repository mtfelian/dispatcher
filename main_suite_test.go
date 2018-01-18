package dispatcher

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

	"github.com/gin-gonic/gin"
	"github.com/mtfelian/dispatcher/counter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	mServer    *mockServer
	dispatcher *Dispatcher
	server     *httptest.Server

	totalFound = counter.New(0)
	maxWorkers = 9
	signals    Channels
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

// startHTTPServer launches the requests router

// apiTreat finds the What on page at given URL
func apiTreat(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var input testInput
	if err := json.Unmarshal(body, &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go func() { signals.data <- input }()
	c.Status(http.StatusOK)
}

func TestApp(t *testing.T) {
	server = httptest.NewServer(func() *gin.Engine {
		router := gin.Default()
		router.POST("/treat", apiTreat)
		return router
	}())
	defer server.Close()

	// init mock server
	mServer = newMockServer()
	defer mServer.Close()

	// init dispatcher
	signals = InitChannels()

	treatFunc := func(element Gi) (Gr, error) {
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

	onResultFunc := func(r Result) {
		n := r.out.(testOutput).n
		fmt.Printf("Count for %s: %d\n", r.in.(testInput).URL, n)
		totalFound.Add(n)
	}

	dispatcher = NewDispatcher(signals, maxWorkers, treatFunc, onResultFunc)
	go dispatcher.Run()

	rand.Seed(time.Now().UnixNano())

	RegisterFailHandler(Fail)
	RunSpecs(t, "Dispatcher Suite")
}

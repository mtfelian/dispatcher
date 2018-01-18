package dispatcher

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
)

// pageAddr contains page route part of URL
const pageAddr = "/page"

// mockServer is a mock server
type mockServer struct{ *httptest.Server }

// imitation is a server route treating imitation
func (server *mockServer) imitation(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.RequestURI, pageAddr) && r.Method == http.MethodGet {
		server.getPageImitation(w, r)
		return
	}
}

// getPageImitation is an imitation for getting the page
func (server *mockServer) getPageImitation(w http.ResponseWriter, r *http.Request) {
	s := regexp.MustCompile(fmt.Sprintf(`%s(\d+)`, pageAddr)).FindStringSubmatch(r.RequestURI)
	if len(s) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "s: %s", s)
		return
	}

	i, err := strconv.ParseInt(s[1], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "expected int, err: %v", err)
		return
	}

	okAnswer := strings.Repeat("11 Go 22 33", int(i))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, okAnswer)
}

// newMockServer returns a pointer to new mock server
func newMockServer() *mockServer {
	var server mockServer
	return &mockServer{Server: httptest.NewServer(http.HandlerFunc((&server).imitation))}
}

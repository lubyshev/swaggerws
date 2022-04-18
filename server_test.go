package swaggerws_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type httpCallback = func(http.ResponseWriter, *http.Request, *responderTester, *sync.WaitGroup)

func server(t *testing.T, cbResponse httpCallback, wg *sync.WaitGroup) (srv *httptest.Server, url string, rTester *responderTester) {
	rTester = &responderTester{t: t}
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cbResponse(w, r, rTester, wg)

	}))
	url = strings.Split(srv.URL, "://")[1]

	return
}

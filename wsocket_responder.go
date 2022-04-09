package swaggerws

import (
	"fmt"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/websocket"
	"net/http"

	"github.com/go-openapi/runtime"
)

type ResponderMiddlewareFunc = func(socket WebSocket, err error) bool

type SocketResponder struct {
	req        *http.Request
	middleware ResponderMiddlewareFunc
}

// NewSocketResponder Returns new SocketResponder.
func NewSocketResponder(req *http.Request, middleware ResponderMiddlewareFunc) middleware.Responder {
	if middleware == nil {
		panic(ErrResponderMiddlewareIsNil)
	}
	return &SocketResponder{
		req:        req,
		middleware: middleware,
	}
}

// WriteResponse Intercepts the response to the client over a websocket.
//
// Also creates a new websocket.
func (wsr *SocketResponder) WriteResponse(rw http.ResponseWriter, _ runtime.Producer) {
	var err error
	var socket WebSocket

	// create a new websocket
	socket, err = NewWebSocket(wsr.req, rw)
	if err != nil {
		wsr.middleware(socket, fmt.Errorf("fail to create websocket: %w", err))
		return
	}

	// try to run middleware
	if !wsr.middleware(socket, nil) {
		if !socket.IsClosed() {
			_ = socket.Close(websocket.CloseInternalServerErr, "internal server error")
		}
		return
	}

	// try to run websocket
	if err = socket.Run(); err != nil {
		wsr.middleware(socket, fmt.Errorf("fail to run websocket client: %w", err))
		if !socket.IsClosed() {
			_ = socket.Close(websocket.CloseInternalServerErr, "internal server error")
		}
		return
	} else {
		panic ("here")
	}
}

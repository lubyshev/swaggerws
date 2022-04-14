package swaggerws_test

import (
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lubyshev/swaggerws"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	msgHello  = "hello!"
	msgClient = "im a client!"
)

func Test_SwaggerResponder_Constructor_FAIL(t *testing.T) {
	defer func() {
		r := recover()
		assert.Equal(t, r, swaggerws.ErrResponderMiddlewareIsNil)
	}()

	_ = swaggerws.NewSocketResponder(nil, nil)
}

func Test_SwaggerResponder_Constructor_OK(t *testing.T) {
	defer func() {
		r := recover()
		assert.Equal(t, nil, r)
	}()

	_ = swaggerws.NewSocketResponder(nil, fakeResponderMiddleware)
}

func Test_SwaggerResponder_WriteResponse(t *testing.T) {
	rt := &responderTester{t: t}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		swagger := swaggerws.NewSocketResponder(r, rt.testResponderMiddleware)
		swagger.WriteResponse(w, nil)
	}))
	defer ts.Close()

	// 1. Invalid connection
	r, _ := http.Get(ts.URL)
	_ = r.Body.Close()

	// prepare web socket connection
	parts := strings.Split(ts.URL, "://")
	wsUrl := "ws://" + parts[1]

	// 2. Invalid middleware answer
	cl := newWsClient()
	cl.Run(wsUrl, rt)

	// 3. Fail to run websocket in the responder
	cl = newWsClient()
	cl.Run(wsUrl, rt)

	// 4. Normal behavior
	cl = newWsClient()
	cl.Run(wsUrl, rt)
}

func fakeResponderMiddleware(swaggerws.WebSocket, error) bool {
	return true
}

type responderTester struct {
	t               *testing.T
	socketID        uuid.UUID
	pool            swaggerws.SocketPool
	middlewareCalls int
	handlerCalls    int
}

func (rt *responderTester) testResponderMiddleware(socket swaggerws.WebSocket, err error) bool {
	rt.middlewareCalls++

	switch rt.middlewareCalls {
	// 1. Invalid connection
	case 1:
		assert.Equal(rt.t,
			"websocket: the client is not using the websocket protocol: 'upgrade' token not found in 'Connection' header",
			errors.Unwrap(errors.Unwrap(err)).Error(),
		)
		return false

	// 2. Invalid middleware answer
	case 2:
		assert.NoError(rt.t, err)
		return false

	// 3.1. Fail to run websocket in the responder
	//      Generate full message stack
	case 3:
		assert.NoError(rt.t, err)
		msg := &swaggerws.WebSocketMessage{
			Type: swaggerws.MsgTypeSocket,
			Msg:  "stack overflow message",
		}
		for i := 0; i < 9; i++ {
			err = socket.WriteToHandler(msg)
		}

	// 3.2. Fail to run websocket in the responder
	//      Check "message stack is overflow" error
	case 4:
		assert.True(rt.t, errors.Is(err, swaggerws.ErrMessageStackOverflow))
		return false

	// 4. Normal behavior
	case 5:
		assert.NoError(rt.t, err)
		id := uuid.New()
		poolID := uuid.New()
		pool := swaggerws.NewSocketPool(poolID)
		socket.
			SetID(id).
			AssignPool(pool).
			AssignHandler(rt.testSocketHandler)
		rt.socketID = socket.GetID()
		rt.pool = socket.Pool()
		assert.Equal(rt.t, id, rt.socketID)
		assert.Equal(rt.t, poolID, rt.pool.GetID())
		s := socket.ResetPool()
		assert.Equal(rt.t, socket, s)
		assert.Equal(rt.t, nil, socket.Pool())
	}

	return true
}

func (rt *responderTester) testSocketHandler(socket swaggerws.WebSocket, err error) {
	var msg *swaggerws.WebSocketMessage
	rt.handlerCalls++

	if err == nil {
		msg = socket.Read()
		assert.Equal(rt.t, (*swaggerws.WebSocketMessage)(nil), socket.Read())
	}
	switch rt.handlerCalls {
	// 3.1 Normal behavior: Init message
	case 1:
		assert.NoError(rt.t, err)
		if msg == nil {
			assert.Fail(rt.t, "message can't be nil")
		} else {
			assert.Equal(rt.t, swaggerws.MsgTypeInit, msg.Type)
		}
		err = socket.Send([]byte(msgHello))
		assert.NoError(rt.t, err)
	case 2:
		assert.NoError(rt.t, err)
		if msg == nil {
			assert.Fail(rt.t, "message can't be nil")
		} else {
			assert.Equal(rt.t, swaggerws.MsgTypeSocket, msg.Type)
			assert.Equal(rt.t, msgClient, string(msg.Msg.([]uint8)))
			err = socket.Close(websocket.CloseNormalClosure, "normal closure")
			assert.NoError(rt.t, err)
			err = socket.Close(websocket.CloseNormalClosure, "normal closure")
			assert.Equal(rt.t, swaggerws.ErrSocketIsClosed, err)
		}
	}
}

func (rt *responderTester) testClientMessages(counter int, conn *websocket.Conn, msg string) {
	switch counter {
	// 4. Normal behavior: Init message
	case 1:
		assert.Equal(rt.t, msgHello, msg)
		err := conn.WriteMessage(websocket.TextMessage, []byte(msgClient))
		assert.NoError(rt.t, err)
	}
}

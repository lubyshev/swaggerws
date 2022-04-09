package swaggerws

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// MsgTypeInit Initialization message (the first message received by the socket handler).
	MsgTypeInit = "swaggerws:message_init"
	// MsgTypeSocket message from the websocket external client via socket.
	MsgTypeSocket = "swaggerws:message_socket"
)

const (
	// Time allowed to write a message to the other side.
	writeWait = 10 * time.Second

	// Maximum message size allowed from the other side.
	maxMessageSize = 1024

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 28 * time.Second
)

type WebSocketMessage struct {
	Type string
	Msg  interface{}
}

type WebSocketHandlerFunc = func(client WebSocket, err error)

type WebSocket interface {
	AssignHandler(handler WebSocketHandlerFunc) WebSocket
	AssignPool(pool SocketPool) WebSocket
	Close(code int, reason string) error
	GetHandler() WebSocketHandlerFunc
	GetID() uuid.UUID
	IsClosed() bool
	Pool() SocketPool
	Read() *WebSocketMessage
	Run() error
	Send(message []byte) error
	SetError(err error)
	SetID(id uuid.UUID) WebSocket
	WriteToHandler(message *WebSocketMessage) error
}

// WebSocketClient is a middleman between the websocket connection and the hub.
type WebSocketClient struct {
	id uuid.UUID

	pool SocketPool

	upgrader websocket.Upgrader
	// The websocket connection.
	conn *websocket.Conn

	handler WebSocketHandlerFunc

	syncContext   context.Context
	syncContextCF context.CancelFunc
	syncWg        *sync.WaitGroup
	runWg         *sync.WaitGroup

	// Buffered channel of outbound messages.
	send chan []byte
	// channel of incoming signals.
	recv chan struct{}
	// Buffered channel of inbound messages.
	messages chan *WebSocketMessage

	// Write to socket mutex.
	mxWrite sync.Mutex
	closed  bool // доступ должен быть защищён, т.к. его читают несколько потоков
	mxClose sync.Mutex

	useWG bool
}

func NewWebSocket(
	req *http.Request,
	rw http.ResponseWriter,
) (WebSocket, error) {
	var err error

	cl := &WebSocketClient{
		send:     make(chan []byte, 8),
		recv:     make(chan struct{}, 8),
		messages: make(chan *WebSocketMessage, 8),
		syncWg:   &sync.WaitGroup{},
		runWg:    &sync.WaitGroup{},
	}

	cl.syncContext, cl.syncContextCF = context.WithCancel(context.Background())

	cl.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	cl.conn, err = cl.upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Err(err).Msg("fail to upgrade connection to websocket")
		cl.syncContextCF()
		return nil, err
	}

	return cl, nil
}

func (c *WebSocketClient) AssignHandler(handler WebSocketHandlerFunc) WebSocket {
	c.handler = handler

	return c
}

func (c *WebSocketClient) AssignPool(pool SocketPool) WebSocket {
	c.pool = pool

	return c
}

func (c *WebSocketClient) GetID() uuid.UUID {
	return c.id
}

func (c *WebSocketClient) GetHandler() WebSocketHandlerFunc {
	return c.handler
}

func (c *WebSocketClient) IsClosed() bool {
	c.mxClose.Lock()
	defer c.mxClose.Unlock()

	return c.closed
}

func (c *WebSocketClient) Pool() SocketPool {
	return c.pool
}

func (c *WebSocketClient) Run() error {
	c.useWG = true
	c.syncWg.Add(1)
	go c.writeSocket(c.syncContext)
	c.syncWg.Add(1)
	go c.messageQueue(c.syncContext)
	c.syncWg.Add(1)
	go c.readSocket(c.syncContext)
	// add one more to prevent exit before ::close()
	c.runWg.Add(1)

	if err := c.WriteToHandler(&WebSocketMessage{MsgTypeInit, nil}); err != nil {
		return err
	}

	c.runWg.Wait()

	return nil
}
func (c *WebSocketClient) SetID(id uuid.UUID) WebSocket {
	c.id = id

	return c
}

// WriteToHandler Emulates reading a message from a socket.
// The message will be put into the received message queue.
// Handler function will be called.
func (c *WebSocketClient) WriteToHandler(message *WebSocketMessage) error {
	defer func() {
		_ = recover()
	}()

	select {
	case c.messages <- message:
	default:
		return ErrMessageStackOverflow
	}
	c.recv <- struct{}{}

	return nil
}

// Send The message will be sent over the websocket.
func (c *WebSocketClient) Send(message []byte) error {
	defer func() {
		_ = recover()
	}()

	select {
	case c.send <- message:
	default:
		return ErrSendStackOverflow
	}

	return nil
}

func (c *WebSocketClient) SetError(err error) {
	if hnd := c.GetHandler(); hnd != nil {
		hnd(c, err)
	}
}

func (c *WebSocketClient) Read() *WebSocketMessage {
	select {
	case b := <-c.messages:
		return b
	default:
	}

	return nil
}

func (c *WebSocketClient) Close(code int, reason string) (err error) {
	c.mxClose.Lock()
	defer func() {
		if !c.closed && c.useWG {
			c.runWg.Done()
			c.closed = true
		}
		c.mxClose.Unlock()
	}()
	if c.closed {
		return ErrSocketIsClosed
	}

	msg := websocket.FormatCloseMessage(code, reason)
	c.mxWrite.Lock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = c.conn.WriteMessage(websocket.CloseMessage, msg)
	c.mxWrite.Unlock()

	c.syncContextCF()
	err = c.conn.Close()
	c.syncWg.Wait()

	close(c.send)
	close(c.messages)
	close(c.recv)

	return err
}

//======================== PRIVATE METHODS ========================

// readSocket pumps messages from the websocket connection to the hub.
//
// The application runs readSocket in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *WebSocketClient) readSocket(ctx context.Context) {
	defer func() {
		//c.log.Debug().Msgf("exit from the readSocket routine for websocket client '%s'", c.ID)
		c.syncWg.Done()
	}()
	//c.log.Debug().Msgf("start readSocket routine for websocket client '%s'", c.ID)

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(data string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msgType, msg, err := c.conn.ReadMessage()
			if _, ok := err.(*websocket.CloseError); ok {
				//c.log.Debug().Err(err).Msg("close connection")
				go c.SetError(fmt.Errorf(
					"closed with: %s: %w",
					err.Error(),
					ErrSocketClosedByClient,
				))
				return
			}

			if err != nil {
				//c.log.Warn().Err(err).
				//	Msg("error reading message from websocket")
				go c.SetError(fmt.Errorf(
					"closed with: %s: %w",
					err.Error(),
					ErrInternalServerError,
				))
				return
			}

			if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
				if len(msg) > 0 {
					//c.log.Debug().
					//	Msgf("message from client: %s", string(msg))
					if err = c.WriteToHandler(&WebSocketMessage{
						Type: MsgTypeSocket,
						Msg:  msg,
					}); err != nil {
						//	c.log.Err(err).
						//		Msgf("fail to send message to the message queue: %s", err.Error())
					}
				}
			} else {
				//c.log.Warn().
				//	Msgf("unsupported message type %d from client: %s", msgType, string(msg))
			}
		}
	}

}

// writeSocket pumps messages from the hub to the websocket connection.
//
// A goroutine running writeSocket is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *WebSocketClient) writeSocket(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		//c.log.Debug().Msgf("exit from the writeSocket routine for websocket client '%s'", c.ID)
		ticker.Stop()
		c.syncWg.Done()
	}()
	//c.log.Debug().Msgf("start writeSocket routine for websocket client '%s'", c.ID)

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-c.send:
			if ok {
				c.mxWrite.Lock()
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				err := c.conn.WriteMessage(websocket.TextMessage, message)
				c.mxWrite.Unlock()
				if err != nil {
					//c.log.Err(err).Msgf("fail to send message: '%s'", string(message))
				}
			}
		case <-ticker.C:
			// send PING message
			c.mxWrite.Lock()
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// fail to get PONG response
				go c.SetError(fmt.Errorf("fail to get PONG response: %w", err))
				return
			}
			// get PONG response
			c.mxWrite.Unlock()
		}
	}
}

func (c *WebSocketClient) messageQueue(ctx context.Context) {
	defer func() {
		//c.log.Debug().Msgf("exit from the messageQueue routine for websocket client '%s'", c.ID)
		c.syncWg.Done()
	}()
	//c.log.Debug().Msgf("start messageQueue routine for websocket client '%s'", c.ID)

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-c.recv:
			if !ok {
				//c.log.Err(ErrMessageQueueRead).Msgf("failed to read from message queue: '%s'", c.ID)
				return
			}
			//c.log.Debug().Msgf("messageQueue got message: '%s'", c.ID)
			if hnd := c.GetHandler(); hnd != nil {
				hnd(c, nil)
			}
		}
	}
}

package swaggerws_test

import (
	"github.com/gorilla/websocket"
	"log"
)

type wsClient struct {
}

func newWsClient() *wsClient {
	return &wsClient{}
}

func (ws *wsClient) Run(url string, rt *responderTester) {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer func() { _ = c.Close() }()

	log.Printf("connected to %v", c.RemoteAddr())

	c.SetPongHandler(func(data string) error {
		return websocket.ErrReadLimit
	})

	done := make(chan struct{}, 1)

	go func() {
		defer close(done)
		readCounter := 0
		for {
			readCounter++
			messageType, message, err := c.ReadMessage()
			if err != nil {
				log.Println("error:", err)
				return
			}
			if messageType == websocket.CloseMessage {
				_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if len(message) > 0 {
				rt.testClientMessages(readCounter, c, string(message))
			}
		}
	}()

	for range done {
		return
	}
}

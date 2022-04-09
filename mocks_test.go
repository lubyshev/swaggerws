package swaggerws_test

import (
	"github.com/google/uuid"
	"github.com/lubyshev/swaggerws"
	"github.com/lubyshev/swaggerws/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func Test_WebSocketMock(t *testing.T) {
	var err error
	id := uuid.New()
	socket := &mocks.WebSocket{}

	socket.On("AssignHandler", mock.AnythingOfType("func(swaggerws.WebSocket, error)")).Return(nil)
	socket.On("AssignPool", mock.AnythingOfType("*swaggerws.socketPoolImplementation")).Return(nil)
	socket.On("Close", mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(nil)
	socket.On("GetHandler").Return(nil)
	socket.On("GetID").Return(id)
	socket.On("IsClosed").Return(false)
	socket.On("Pool").Return(nil)
	socket.On("Read").Return(&swaggerws.WebSocketMessage{Msg: "mock test message"})
	socket.On("Run").Return(nil)
	socket.On("Send", mock.AnythingOfType("[]uint8")).Return(nil)
	socket.On("SetError", nil).Return()
	socket.On("SetID", mock.AnythingOfType("uuid.UUID")).Return(nil)
	socket.On("WriteToHandler", mock.AnythingOfType("*swaggerws.WebSocketMessage")).Return(nil)

	s := socket.AssignHandler(mockHandler)
	assert.Equal(t, nil, s)

	s = socket.AssignPool(swaggerws.NewSocketPool(uuid.New()))
	assert.Equal(t, nil, s)

	err = socket.Close(1, "")
	assert.Equal(t, nil, err)

	h := socket.GetHandler()
	assert.True(t, h == nil)

	sid := socket.GetID()
	assert.Equal(t, id, sid)

	b := socket.IsClosed()
	assert.Equal(t, false, b)

	p := socket.Pool()
	assert.Equal(t, nil, p)

	r := socket.Read()
	assert.Equal(t, "mock test message", r.Msg)

	err = socket.Run()
	assert.Equal(t, nil, err)

	bt := make([]uint8, 0)
	err = socket.Send(bt)
	assert.Equal(t, nil, err)

	socket.SetError(nil)

	s = socket.SetID(uuid.New())
	assert.Equal(t, nil, s)

	err = socket.WriteToHandler(&swaggerws.WebSocketMessage{})
	assert.Equal(t, nil, err)
}

func mockHandler(swaggerws.WebSocket, error) {}

// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	swaggerws "github.com/lubyshev/swaggerws"
	mock "github.com/stretchr/testify/mock"

	uuid "github.com/google/uuid"
)

// WebSocket is an autogenerated mock type for the WebSocket type
type WebSocket struct {
	mock.Mock
}

// AssignHandler provides a mock function with given fields: handler
func (_m *WebSocket) AssignHandler(handler func(swaggerws.WebSocket, error)) swaggerws.WebSocket {
	return nil
}

// AssignPool provides a mock function with given fields: pool
func (_m *WebSocket) AssignPool(pool swaggerws.SocketPool) swaggerws.WebSocket {
	return nil
}

// Close provides a mock function with given fields: code, reason
func (_m *WebSocket) Close(code int, reason string) error {
	ret := _m.Called(code, reason)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, string) error); ok {
		r0 = rf(code, reason)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetHandler provides a mock function with given fields:
func (_m *WebSocket) GetHandler() func(swaggerws.WebSocket, error) {
	return nil
}

// GetID provides a mock function with given fields:
func (_m *WebSocket) GetID() uuid.UUID {
	ret := _m.Called()

	var r0 uuid.UUID
	if rf, ok := ret.Get(0).(func() uuid.UUID); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(uuid.UUID)
		}
	}

	return r0
}

// IsClosed provides a mock function with given fields:
func (_m *WebSocket) IsClosed() bool {
	return false
}

// Pool provides a mock function with given fields:
func (_m *WebSocket) Pool() swaggerws.SocketPool {
	return nil
}

// Read provides a mock function with given fields:
func (_m *WebSocket) Read() *swaggerws.WebSocketMessage {
	return nil
}

// Run provides a mock function with given fields:
func (_m *WebSocket) Run() error {
	return nil
}

// Send provides a mock function with given fields: message
func (_m *WebSocket) Send(message []byte) error {
	return nil
}

// SetError provides a mock function with given fields: err
func (_m *WebSocket) SetError(err error) {
	_m.Called(err)
}

// SetID provides a mock function with given fields: id
func (_m *WebSocket) SetID(id uuid.UUID) swaggerws.WebSocket {
	return nil
}

// WriteToHandler provides a mock function with given fields: message
func (_m *WebSocket) WriteToHandler(message *swaggerws.WebSocketMessage) error {
	return nil
}
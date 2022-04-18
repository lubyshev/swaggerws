package swaggerws

import "errors"

var (
	ErrInternalServerError = errors.New("internal server error")
	ErrManagerDestroyed    = errors.New("socket manager is destroyed")

	ErrMessageStackOverflow = errors.New("socket message-stack is overflow")
	ErrSendStackOverflow    = errors.New("socket send-stack is overflow")

	ErrSocketIsClosed       = errors.New("socket is already closed")
	ErrSocketClosedByClient = errors.New("socket closed by client from other side")

	ErrPoolAlreadyInContainer = errors.New("socket pool already exists in the container")
	ErrPoolNotFoundInContainer = errors.New("socket pool is not found in the container")

	ErrSocketAlreadyInPool    = errors.New("socket already exists in the pool")
	ErrSocketNotFoundInPool    = errors.New("socket is not found in the pool")

	ErrResponderMiddlewareIsNil = errors.New("responder middleware function is nil")
)

package swaggerws

import (
	"github.com/google/uuid"
	"sync"
	"sync/atomic"
)

type SocketPool interface {
	AllSocketsCount() int64
	AppendPool(SocketPool) error
	AppendSocket(WebSocket) error
	AssignSocketManager(SocketManager) SocketPool
	Clear()
	GetAllSockets() <-chan WebSocket
	GetID() uuid.UUID
	GetPoolByID(uuid.UUID) (SocketPool, error)
	GetSocketByID(uuid.UUID) (WebSocket, error)
	PoolsCount() int64
	PoolsRange() <-chan SocketPool
	SocketsCount() int64
	SocketManager() SocketManager
	SocketsRange() <-chan WebSocket
}

type socketPoolImplementation struct {
	id            uuid.UUID
	sockets       sync.Map
	socketManager SocketManager
	socketsCount  int64
	pools         sync.Map
	poolsCount    int64
}

func NewSocketPool(socketId uuid.UUID) SocketPool {
	return &socketPoolImplementation{
		id: socketId,
	}
}

func (si *socketPoolImplementation) AllSocketsCount() (cnt int64) {
	if si.poolsCount == 0 && si.socketsCount == 0 {
		return
	}

	for p := range si.PoolsRange() {
		cnt += p.AllSocketsCount()
	}

	return cnt + si.socketsCount
}

func (si *socketPoolImplementation) AppendPool(pool SocketPool) error {
	if _, loaded := si.pools.LoadOrStore(pool.GetID(), pool); loaded {
		return ErrPoolAlreadyInContainer
	}
	atomic.AddInt64(&si.poolsCount, 1)

	return nil
}

func (si *socketPoolImplementation) AppendSocket(socket WebSocket) error {
	if _, loaded := si.sockets.LoadOrStore(socket.GetID(), socket); loaded {
		return ErrSocketAlreadyInPool
	}
	atomic.AddInt64(&si.socketsCount, 1)

	return nil
}

// AssignSocketManager Appends init function to the SocketResponder.
func (si *socketPoolImplementation) AssignSocketManager(socketManager SocketManager) SocketPool {
	si.socketManager = socketManager

	return si
}

func (si *socketPoolImplementation) GetAllSockets() <-chan WebSocket {
	ch := make(chan WebSocket)
	go func() {
		for s := range si.SocketsRange() {
			ch <- s
		}
		for p := range si.PoolsRange() {
			for s := range p.GetAllSockets() {
				ch <- s
			}
		}
		close(ch)
	}()

	return ch
}

func (si *socketPoolImplementation) GetID() uuid.UUID {
	return si.id
}

func (si *socketPoolImplementation) Clear() {
	si.pools.Range(func(key, value interface{}) bool {
		value.(SocketPool).Clear()
		si.pools.Delete(key)
		atomic.AddInt64(&si.poolsCount, -1)
		return true
	})
	si.sockets.Range(func(key, value interface{}) bool {
		si.sockets.Delete(key)
		atomic.AddInt64(&si.socketsCount, -1)
		return true
	})
}

func (si *socketPoolImplementation) GetPoolByID(id uuid.UUID) (SocketPool, error) {
	if p, ok := si.pools.Load(id); ok {
		return p.(SocketPool), nil
	}

	return nil, ErrPoolNotFoundInContainer
}

func (si *socketPoolImplementation) GetSocketByID(id uuid.UUID) (WebSocket, error) {
	if p, ok := si.sockets.Load(id); ok {
		return p.(WebSocket), nil
	}

	return nil, ErrSocketNotFoundInPool
}

func (si *socketPoolImplementation) PoolsCount() int64 {
	return si.poolsCount
}

func (si *socketPoolImplementation) PoolsRange() <-chan SocketPool {
	ch := make(chan SocketPool)
	go func() {
		si.pools.Range(func(key, value interface{}) bool {
			ch <- value.(SocketPool)
			return true
		})
		close(ch)
	}()

	return ch
}

func (si *socketPoolImplementation) SocketsCount() int64 {
	return si.socketsCount
}

func (si *socketPoolImplementation) SocketManager() SocketManager {
	return si.socketManager
}

func (si *socketPoolImplementation) SocketsRange() <-chan WebSocket {
	ch := make(chan WebSocket)
	go func() {
		si.sockets.Range(func(key, value interface{}) bool {
			ch <- value.(WebSocket)
			return true
		})
		close(ch)
	}()

	return ch
}

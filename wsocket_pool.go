package swaggerws

import (
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

type SocketPool interface {
	AllSocketCount() int64
	AppendPool(SocketPool) error
	AppendSocket(WebSocket) error
	AssignSocketManager(SocketManager) SocketPool
	Clear()
	DeletePool(id uuid.UUID) error
	DeleteSocket(id uuid.UUID) error
	GetAllSockets() <-chan WebSocket
	GetID() uuid.UUID
	GetPool(uuid.UUID) (SocketPool, error)
	GetSocket(uuid.UUID) (WebSocket, error)
	Manager() SocketManager
	PoolsCount() int64
	PoolRange() <-chan SocketPool
	ResetManager() SocketPool
	SocketsCount() int64
	SocketRange() <-chan WebSocket
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

func (si *socketPoolImplementation) AllSocketCount() (cnt int64) {
	if si.poolsCount == 0 && si.socketsCount == 0 {
		return
	}

	for p := range si.PoolRange() {
		cnt += p.AllSocketCount()
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
		for s := range si.SocketRange() {
			ch <- s
		}
		for p := range si.PoolRange() {
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
		value.(WebSocket).ResetPool()
		si.sockets.Delete(key)
		atomic.AddInt64(&si.socketsCount, -1)
		return true
	})
}

func (si *socketPoolImplementation) DeletePool(id uuid.UUID) error {
	if p, err := si.GetPool(id); err != nil {
		return err
	} else {
		p.Clear()
	}
	si.pools.Delete(id)
	atomic.AddInt64(&si.poolsCount, -1)

	return nil
}

func (si *socketPoolImplementation) DeleteSocket(id uuid.UUID) error {
	if s, err := si.GetSocket(id); err != nil {
		return err
	} else {
		s.ResetPool()
	}
	si.sockets.Delete(id)
	atomic.AddInt64(&si.socketsCount, -1)

	return nil
}

func (si *socketPoolImplementation) GetPool(id uuid.UUID) (SocketPool, error) {
	if p, ok := si.pools.Load(id); ok {
		return p.(SocketPool), nil
	}

	return nil, ErrPoolNotFoundInContainer
}

func (si *socketPoolImplementation) GetSocket(id uuid.UUID) (WebSocket, error) {
	if p, ok := si.sockets.Load(id); ok {
		return p.(WebSocket), nil
	}

	return nil, ErrSocketNotFoundInPool
}

func (si *socketPoolImplementation) Manager() SocketManager {
	return si.socketManager
}

func (si *socketPoolImplementation) PoolsCount() int64 {
	return atomic.LoadInt64(&si.poolsCount)
}

func (si *socketPoolImplementation) PoolRange() <-chan SocketPool {
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

func (si *socketPoolImplementation) ResetManager() SocketPool {
	si.socketManager = nil

	return si
}

func (si *socketPoolImplementation) SocketsCount() int64 {
	return atomic.LoadInt64(&si.socketsCount)
}

func (si *socketPoolImplementation) SocketRange() <-chan WebSocket {
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

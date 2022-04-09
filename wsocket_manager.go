package swaggerws

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"sync"
	"sync/atomic"
)

const (
	managerStatusReady = iota
	managerStatusDestroyed
)

type SocketManager interface {
	AppendPool(pool SocketPool) error
	Destroy() chan error
	GetOrCreatePool(poolID uuid.UUID) (SocketPool, error)
	GetPoolById(poolID uuid.UUID) (SocketPool, error)
	IsDestroyed() bool
	IsReady() bool
}

type socketManagerImplementation struct {
	mxPools sync.RWMutex
	pool    SocketPool
	status  int32
}

// NewSocketManager Returns a new websocket client manager.
func NewSocketManager() SocketManager {
	return &socketManagerImplementation{
		pool:   NewSocketPool(uuid.New()),
		status: managerStatusReady,
	}
}

func (smi *socketManagerImplementation) AppendPool(pool SocketPool) error {
	if smi.IsDestroyed() {
		return ErrManagerDestroyed
	}
	smi.mxPools.Lock()
	defer func() {
		smi.mxPools.Unlock()
	}()

	return smi.appendPool(pool)
}

func (smi *socketManagerImplementation) appendPool(pool SocketPool) error {
	if smi.IsDestroyed() {
		return ErrManagerDestroyed
	}
	return smi.pool.AppendPool(pool)
}

func (smi *socketManagerImplementation) Destroy() (chErrors chan error) {
	smi.mxPools.Lock()
	defer smi.mxPools.Unlock()

	// if instance is destroyed
	if smi.IsDestroyed() {
		chErrors = make(chan error, 1)
		chErrors <- ErrManagerDestroyed
		close(chErrors)
	} else {
		smi.markAsDestroyed()
		chErrors = make(chan error, smi.pool.AllSocketsCount())
		go func() {
			for socket := range smi.pool.GetAllSockets() {
				if err := socket.Close(websocket.CloseGoingAway, "server shutdown"); err != nil {
					chErrors <- err
				}
			}
			for pool := range smi.pool.PoolsRange() {
				pool.Clear()
			}
			close(chErrors)
		}()
	}

	return
}

func (smi *socketManagerImplementation) GetOrCreatePool(poolID uuid.UUID) (pool SocketPool, err error) {
	if smi.IsDestroyed() {
		return nil, ErrManagerDestroyed
	}
	smi.mxPools.Lock()
	defer smi.mxPools.Unlock()

	pool, err = smi.getPoolById(poolID)
	if err == ErrPoolNotFoundInContainer {
		pool = NewSocketPool(poolID)
		err = smi.appendPool(pool)
		pool.AssignSocketManager(smi)
	}

	return
}

func (smi *socketManagerImplementation) GetPoolById(poolID uuid.UUID) (SocketPool, error) {
	if smi.IsDestroyed() {
		return nil, ErrManagerDestroyed
	}
	smi.mxPools.Lock()
	defer smi.mxPools.Unlock()

	return smi.getPoolById(poolID)
}

func (smi *socketManagerImplementation) getPoolById(poolID uuid.UUID) (SocketPool, error) {
	if smi.IsDestroyed() {
		return nil, ErrManagerDestroyed
	}
	if !smi.IsReady() {
		return nil, ErrManagerDestroyed
	}

	return smi.pool.GetPoolByID(poolID)
}

func (smi *socketManagerImplementation) IsReady() bool {
	return atomic.LoadInt32(&smi.status) == managerStatusReady
}

func (smi *socketManagerImplementation) markAsDestroyed() {
	atomic.StoreInt32(&smi.status, managerStatusDestroyed)
}

func (smi *socketManagerImplementation) IsDestroyed() bool {
	return atomic.LoadInt32(&smi.status) == managerStatusDestroyed
}

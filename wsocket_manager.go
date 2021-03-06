package swaggerws

import (
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	managerStatusReady = iota
	managerStatusDestroyed
)

type SocketManager interface {
	AppendPool(pool SocketPool) error
	Destroy() chan error
	GetOrCreatePool(poolID uuid.UUID) (SocketPool, bool, error)
	GetPool(poolID uuid.UUID) (SocketPool, error)
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
	defer smi.mxPools.Unlock()

	return smi.appendPool(pool)
}

func (smi *socketManagerImplementation) appendPool(pool SocketPool) error {
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
		chErrors = make(chan error, smi.pool.AllSocketCount())
		go func() {
			for socket := range smi.pool.GetAllSockets() {
				if !socket.IsClosed() {
					err := socket.Close(websocket.CloseGoingAway, "server shutdown");
					if err != nil {
						chErrors <- err
					}
				}
			}
			for pool := range smi.pool.PoolRange() {
				pool.Clear()
			}
			close(chErrors)
		}()
	}

	return
}

func (smi *socketManagerImplementation) GetOrCreatePool(poolID uuid.UUID) (pool SocketPool, isNew bool, err error) {
	if smi.IsDestroyed() {
		return nil, false, ErrManagerDestroyed
	}
	smi.mxPools.Lock()
	defer smi.mxPools.Unlock()

	pool, err = smi.getPoolById(poolID)
	if err == ErrPoolNotFoundInContainer {
		pool = NewSocketPool(poolID)
		err = smi.appendPool(pool)
		pool.AssignSocketManager(smi)
		isNew = true
	}

	return
}

func (smi *socketManagerImplementation) GetPool(poolID uuid.UUID) (SocketPool, error) {
	if smi.IsDestroyed() {
		return nil, ErrManagerDestroyed
	}
	smi.mxPools.Lock()
	defer smi.mxPools.Unlock()

	return smi.getPoolById(poolID)
}

func (smi *socketManagerImplementation) getPoolById(poolID uuid.UUID) (SocketPool, error) {
	return smi.pool.GetPool(poolID)
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

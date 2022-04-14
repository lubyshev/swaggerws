package swaggerws_test

import (
	"github.com/google/uuid"
	"github.com/lubyshev/swaggerws"
	"github.com/lubyshev/swaggerws/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/rand"
	"testing"
)

func Test_SocketPool(t *testing.T) {

	t.Run("test AppendSocket() method", func(t *testing.T) {
		var err error
		pool := swaggerws.NewSocketPool(uuid.New())

		socketID := uuid.New()
		socket := &mocks.WebSocket{}
		socket.On("GetID").Return(socketID)
		assert.Equal(t, socketID, socket.GetID())
		socket.On(
			"AssignPool", mock.AnythingOfType("*swaggerws.socketPoolImplementation"),
		).Return(socket)
		socket.On("ResetPool").Return(socket)

		socket2ID := uuid.New()
		socket2 := &mocks.WebSocket{}
		socket2.On("GetID").Return(socket2ID)
		assert.Equal(t, socket2ID, socket2.GetID())

		var s swaggerws.WebSocket

		err = pool.AppendSocket(socket)
		assert.NoError(t, err)
		s, err = pool.GetSocket(socketID)
		assert.NoError(t, err)
		assert.Equal(t, socket, s)
		err = pool.AppendSocket(socket)
		assert.Equal(t, swaggerws.ErrSocketAlreadyInPool, err)
		assert.Equal(t, socket, s)

		err = pool.AppendSocket(socket2)
		assert.NoError(t, err)
		s, err = pool.GetSocket(socket2ID)
		assert.NoError(t, err)
		assert.Equal(t, socket2, s)
		err = pool.AppendSocket(socket2)
		assert.Equal(t, swaggerws.ErrSocketAlreadyInPool, err)
		assert.Equal(t, socket2, s)

		s, err = pool.GetSocket(uuid.New())
		assert.Equal(t, swaggerws.ErrSocketNotFoundInPool, err)
		assert.Equal(t, nil, s)

		assert.Equal(t, int64(2), pool.SocketsCount())
		assert.Equal(t, int64(2), pool.AllSocketsCount())

		err = pool.DeleteSocket(socketID)
		assert.NoError(t, err)
		s, err = pool.GetSocket(uuid.New())
		assert.Equal(t, swaggerws.ErrSocketNotFoundInPool, err)
		assert.Equal(t, nil, s)
		err = pool.DeleteSocket(socketID)
		assert.Equal(t, swaggerws.ErrSocketNotFoundInPool, err)
	})

	t.Run("test AppendPool() method", func(t *testing.T) {
		var err error
		pool := swaggerws.NewSocketPool(uuid.New())

		var pp swaggerws.SocketPool
		var pCnt int64
		var lsCnt int64
		var id uuid.UUID
		sCnt := pool.AllSocketsCount()
		for pCnt = 0; pCnt < int64(2+rand.Intn(5)); pCnt++ {
			id = uuid.New()
			p, cnt, _ := getNewPool(id)
			lsCnt = cnt
			sCnt += cnt

			err = pool.AppendPool(p)
			assert.NoError(t, err)
			pp, err = pool.GetPool(id)
			assert.NoError(t, err)
			assert.Equal(t, p, pp)
			err = pool.AppendPool(p)
			assert.Equal(t, swaggerws.ErrPoolAlreadyInContainer, err)
			assert.Equal(t, p, pp)

			pp, err = pool.GetPool(uuid.New())
			assert.Equal(t, swaggerws.ErrPoolNotFoundInContainer, err)
			assert.Equal(t, nil, pp)

			assert.Equal(t, cnt, p.SocketsCount())
		}
		err = pool.DeletePool(id)
		assert.NoError(t, err)
		err = pool.DeletePool(id)
		assert.Equal(t, swaggerws.ErrPoolNotFoundInContainer, err)

		assert.Equal(t, sCnt-lsCnt, pool.AllSocketsCount())
		assert.Equal(t, pCnt-1, pool.PoolsCount())
	})

	t.Run("test Range() methods", func(t *testing.T) {
		pool := swaggerws.NewSocketPool(uuid.New())
		sockets := make(map[uuid.UUID]swaggerws.WebSocket)
		pools := make(map[uuid.UUID]swaggerws.SocketPool)

		var pCnt int64
		sCnt := pool.AllSocketsCount()
		for pCnt = 0; pCnt < int64(2+rand.Intn(5)); pCnt++ {
			id := uuid.New()
			p, cnt, ss := getNewPool(id)
			sCnt += cnt
			for key, value := range ss { // Order not specified
				sockets[key] = value
			}
			_ = pool.AppendPool(p)
			pools[p.GetID()] = p
			var sip int64
			for s := range p.SocketsRange() {
				if sm, ok := sockets[s.GetID()]; ok {
					sip++
					assert.Equal(t, sm, s)
				} else {
					assert.Fail(t, "socket not found in the pool")
				}
			}
			assert.Equal(t, cnt, sip)
		}

		var pip int64
		for p := range pool.PoolsRange() {
			if pm, ok := pools[p.GetID()]; ok {
				pip++
				assert.Equal(t, pm, p)
			} else {
				assert.Fail(t, "pool not found in the container")
			}
		}
		assert.Equal(t, pCnt, pip)

		var sip int64
		for s := range pool.GetAllSockets() {
			if sm, ok := sockets[s.GetID()]; ok {
				sip++
				assert.Equal(t, sm, s)
			} else {
				assert.Fail(t, "socket not found in the main pool")
			}
		}
		assert.Equal(t, sCnt, sip)
	})

	t.Run("test Clear() method", func(t *testing.T) {
		pool := swaggerws.NewSocketPool(uuid.New())
		pools := make(map[uuid.UUID]swaggerws.SocketPool)

		var pCnt int64
		sCnt := pool.AllSocketsCount()
		for pCnt = 0; pCnt < int64(2+rand.Intn(5)); pCnt++ {
			id := uuid.New()
			p, cnt, _ := getNewPool(id)
			sCnt += cnt
			_ = pool.AppendPool(p)
			pools[p.GetID()] = p
		}
		assert.Equal(t, sCnt, pool.AllSocketsCount())
		assert.Equal(t, pCnt, pool.PoolsCount())

		pool.Clear()
		assert.Equal(t, int64(0), pool.SocketsCount())
		assert.Equal(t, int64(0), pool.AllSocketsCount())
		assert.Equal(t, int64(0), pool.PoolsCount())
		for _, p := range pools {
			assert.Equal(t, int64(0), p.SocketsCount())
			assert.Equal(t, int64(0), p.AllSocketsCount())
			assert.Equal(t, int64(0), p.PoolsCount())
		}
	})

	t.Run("test AssignSocketManager() method", func(t *testing.T) {
		pool := swaggerws.NewSocketPool(uuid.New())

		s := pool.SocketManager()
		assert.Equal(t, nil, s)

		sm := swaggerws.NewSocketManager()
		p := pool.AssignSocketManager(sm)
		assert.Equal(t, pool, p)

		s = pool.SocketManager()
		assert.Equal(t, sm, s)
	})
}

func getNewPool(id uuid.UUID) (swaggerws.SocketPool, int64, map[uuid.UUID]swaggerws.WebSocket) {
	pool := swaggerws.NewSocketPool(id)

	var i int64
	sockets := make(map[uuid.UUID]swaggerws.WebSocket)
	for i = 0; i < int64(2+rand.Intn(5)); i++ {
		socketID := uuid.New()
		socket := &mocks.WebSocket{}
		socket.On("GetID").Return(socketID)
		var err error
		if i%2 == 1 {
			err = swaggerws.ErrInternalServerError
		}
		socket.On(
			"Close",
			mock.AnythingOfType("int"),
			mock.AnythingOfType("string"),
		).Return(err)
		socket.On(
			"AssignPool", mock.AnythingOfType("*swaggerws.socketPoolImplementation"),
		).Return(socket)
		socket.On("ResetPool").Return(socket)
		sockets[socketID] = socket
		_ = pool.AppendSocket(socket)
	}

	return pool, i, sockets
}

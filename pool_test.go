package swaggerws_test

import (
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lubyshev/swaggerws"
	"github.com/stretchr/testify/assert"
)

func Test_SocketPool(t *testing.T) {
	t.Run("test Socket() methods", func(t *testing.T) {
		ts, url, rt := server(t, cbPoolTestSocket, nil)
		defer ts.Close()
		url = "ws://" + url
		cl := newWsClient()
		cl.Run(url, rt)
	})

	t.Run("test Pool() methods", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		ts, url, rt := server(t, cbPoolTestPool, wg)
		defer ts.Close()

		rt.pool = swaggerws.NewSocketPool(uuid.New())
		url = "ws://" + url

		// fill the root pool with children
		for i := 0; i < 3; i++ {
			cl := newWsClient()
			cl.Run(url, rt)
		}
		wg.Wait()

		// test pool range methods
		cl := newWsClient()
		cl.Run(url, rt)

		wg.Wait()
	})
}

func cbPoolTestSocket(w http.ResponseWriter, rq *http.Request, rt *responderTester, _ *sync.WaitGroup) {
	socketID := uuid.New()
	socket, err := swaggerws.NewWebSocket(rq, w)
	defer func() {
		_ = socket.Close(websocket.CloseNormalClosure, "test AppendSocket() callback finished")
	}()
	assert.NoError(rt.t, err)
	assert.Equal(rt.t, socket, socket.SetID(socketID))
	assert.Equal(rt.t, socketID, socket.GetID())
	assertAppendSocket(rt.t, socketID, socket)
}

func cbPoolTestPool(w http.ResponseWriter, rq *http.Request, rt *responderTester, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)

	socket, err := swaggerws.NewWebSocket(rq, w)
	assert.NoError(rt.t, err)
	socket.SetID(uuid.New())

	rt.middlewareCalls++
	switch rt.middlewareCalls {
	case 1, 2, 3:
		defer func() { _ = socket.Close(websocket.CloseNormalClosure, "test AppendPool():fill callback finished") }()
		assertAppendPool(socket, rt)
	case 4:
		defer func() { _ = socket.Close(websocket.CloseNormalClosure, "test AppendPool():range callback finished") }()
		assertPoolRange(socket, rt)
	default:
		defer func() { _ = socket.Close(websocket.CloseAbnormalClosure, "error: test pool callback: invalid middlewareCalls value") }()
		panic(errors.New("error: test pool callback: invalid middlewareCalls value"))
	}
}

func assertAppendSocket(t *testing.T, socketID uuid.UUID, socket swaggerws.WebSocket) {
	pool := swaggerws.NewSocketPool(uuid.New())
	err := pool.AppendSocket(socket)
	assert.NoError(t, err)

	s, err := pool.GetSocket(socketID)
	assert.NoError(t, err)
	assert.Equal(t, socket, s)

	err = pool.AppendSocket(socket)
	assert.Equal(t, swaggerws.ErrSocketAlreadyInPool, err)
	assert.Equal(t, socket, s)

	s, err = pool.GetSocket(uuid.New())
	assert.Equal(t, swaggerws.ErrSocketNotFoundInPool, err)
	assert.Equal(t, nil, s)

	assert.Equal(t, int64(1), pool.SocketsCount())
	assert.Equal(t, int64(1), pool.AllSocketCount())

	err = pool.DeleteSocket(socketID)
	assert.NoError(t, err)
	s, err = pool.GetSocket(uuid.New())
	assert.Equal(t, swaggerws.ErrSocketNotFoundInPool, err)
	assert.Equal(t, nil, s)
	err = pool.DeleteSocket(socketID)
	assert.Equal(t, swaggerws.ErrSocketNotFoundInPool, err)

	assert.Equal(t, int64(0), pool.SocketsCount())
	assert.Equal(t, int64(0), pool.AllSocketCount())
}

func assertAppendPool(socket swaggerws.WebSocket, rt *responderTester) {
	var pp swaggerws.SocketPool
	var t = rt.t

	poolID := uuid.New()
	pool := swaggerws.NewSocketPool(poolID)
	err := pool.AppendSocket(socket)
	assert.NoError(t, err)

	err = rt.pool.AppendPool(pool)
	assert.NoError(t, err)
	pp, err = rt.pool.GetPool(poolID)
	assert.NoError(t, err)
	assert.Equal(t, pool, pp)

	err = rt.pool.AppendPool(pool)
	assert.Equal(t, swaggerws.ErrPoolAlreadyInContainer, err)

	pp, err = rt.pool.GetPool(uuid.New())
	assert.Equal(t, swaggerws.ErrPoolNotFoundInContainer, err)
	assert.Equal(t, nil, pp)
}

func assertPoolRange(socket swaggerws.WebSocket, rt *responderTester) {
	var err error
	var pCnt int64
	var sCnt int64
	var t = rt.t

	for p := range rt.pool.PoolRange() {
		newPool := swaggerws.NewSocketPool(uuid.New())
		err = newPool.AppendSocket(socket)
		assert.NoError(t, err)
		err = p.AppendPool(newPool)
		assert.NoError(t, err)
		sCnt += p.AllSocketCount()
		pCnt++
	}
	assert.Equal(t, pCnt, rt.pool.PoolsCount())
	assert.Equal(t, sCnt, rt.pool.AllSocketCount())

	sCnt = 0
	for range rt.pool.GetAllSockets() {
		sCnt++
	}
	assert.Equal(t, rt.pool.AllSocketCount(), sCnt)

	for p := range rt.pool.PoolRange() {
		var asCnt int64
		for range p.SocketRange() {
			asCnt++
		}
		assert.Equal(t, int64(1), asCnt)

		err = rt.pool.DeletePool(p.GetID())
		assert.NoError(t, err)
		p.Clear()
		assert.Equal(t, int64(0), p.PoolsCount())
		assert.Equal(t, int64(0), p.AllSocketCount())
	}

	id := rt.pool.GetID()
	rt.pool.Clear()
	assert.Equal(t, int64(0), rt.pool.PoolsCount())
	assert.Equal(t, int64(0), rt.pool.AllSocketCount())
	err = rt.pool.DeletePool(id)
	assert.Equal(t, swaggerws.ErrPoolNotFoundInContainer, err)
}

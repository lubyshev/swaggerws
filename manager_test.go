package swaggerws_test

import (
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lubyshev/swaggerws"
	"github.com/stretchr/testify/assert"
	"net/http"
	"sync"
	"testing"
)

func Test_SocketManager(t *testing.T) {

	t.Run("test AppendPool() method", func(t *testing.T) {
		var err error

		m := swaggerws.NewSocketManager()
		id := uuid.New()
		p := swaggerws.NewSocketPool(id)

		err = m.AppendPool(p)
		assert.NoError(t, err)
		err = m.AppendPool(p)
		assert.Equal(t, swaggerws.ErrPoolAlreadyInContainer, err)

		pp, err := m.GetPool(id)
		assert.NoError(t, err)
		assert.Equal(t, p, pp)

		_, err = m.GetPool(uuid.New())
		assert.Equal(t, swaggerws.ErrPoolNotFoundInContainer, err)
	})

	t.Run("test GetOrCreatePool() method", func(t *testing.T) {
		var err error

		m := swaggerws.NewSocketManager()
		id := uuid.New()

		p, created, err := m.GetOrCreatePool(id)
		assert.NoError(t, err)
		assert.True(t, created)
		assert.Equal(t, m, p.Manager())
		err = m.AppendPool(p)
		assert.Equal(t, swaggerws.ErrPoolAlreadyInContainer, err)
		pp, created, err := m.GetOrCreatePool(id)
		assert.NoError(t, err)
		assert.False(t, created)
		assert.Equal(t, p, pp)

		pp, err = m.GetPool(id)
		assert.NoError(t, err)
		assert.Equal(t, p, pp)

		p.ResetManager()
		assert.Equal(t, nil, p.Manager())
	})

	t.Run("test Destroy() method", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		ts, url, rt := server(t, cbManagerTest, wg)
		defer ts.Close()
		url = "ws://" + url

		rt.manager = swaggerws.NewSocketManager()

		for i := 0; i < 3; i++ {
			cl := newWsClient()
			cl.Run(url, rt)
		}
		wg.Wait()

		cl := newWsClient()
		cl.Run(url, rt)
		wg.Wait()

	})
}

func cbManagerTest(w http.ResponseWriter, rq *http.Request, rt *responderTester, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)

	socket, err := swaggerws.NewWebSocket(rq, w)
	assert.NoError(rt.t, err)

	socket.SetID(uuid.New())

	rt.middlewareCalls++
	switch rt.middlewareCalls {
	case 1, 2, 3:
		defer func() { _ = socket.Close(websocket.CloseNormalClosure, "test Manager():fill callback finished") }()
		assertFillManager(socket, rt)
	case 4:
		// socket must be closed by SocketManager::Destroy()
		// inside assertManagerDestroy() function
		assertManagerDestroy(socket, rt)
	default:
		defer func() { _ = socket.Close(websocket.CloseAbnormalClosure, "error: test manager callback: invalid middlewareCalls value") }()
		panic(errors.New("error: test manager callback: invalid middlewareCalls value"))
	}
}

func assertFillManager(socket swaggerws.WebSocket, rt *responderTester) {
	var t = rt.t

	p := swaggerws.NewSocketPool(uuid.New())
	_ = p.AppendSocket(socket)
	err := rt.manager.AppendPool(p)
	assert.NoError(t, err)

	assert.Equal(t, true, rt.manager.IsReady())
	assert.Equal(t, false, rt.manager.IsDestroyed())
}

func assertManagerDestroy(socket swaggerws.WebSocket, rt *responderTester) {
	var t = rt.t
	var m = rt.manager

	p := swaggerws.NewSocketPool(uuid.New())
	_ = p.AppendSocket(socket)
	err := m.AppendPool(p)
	assert.NoError(t, err)

	for err = range m.Destroy() {
		panic(err)
	}
	assert.Equal(t, false, m.IsReady())
	assert.Equal(t, true, m.IsDestroyed())

	errs := m.Destroy()

	assert.Equal(t, 1, len(errs))
	assert.Equal(t, swaggerws.ErrManagerDestroyed, <-errs)

	_, err = m.GetPool(uuid.New())
	assert.Equal(t, err, swaggerws.ErrManagerDestroyed)

	_, _, err = m.GetOrCreatePool(uuid.New())
	assert.Equal(t, err, swaggerws.ErrManagerDestroyed)

	err = m.AppendPool(nil)
	assert.Equal(t, err, swaggerws.ErrManagerDestroyed)
}

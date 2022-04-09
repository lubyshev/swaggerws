package swaggerws_test

import (
	"github.com/google/uuid"
	"github.com/lubyshev/swaggerws"
	"github.com/stretchr/testify/assert"
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

		pp, err := m.GetPoolById(id)
		assert.NoError(t, err)
		assert.Equal(t, p, pp)

		_, err = m.GetPoolById(uuid.New())
		assert.Equal(t, swaggerws.ErrPoolNotFoundInContainer, err)
	})

	t.Run("test GetOrCreatePool() method", func(t *testing.T) {
		var err error

		m := swaggerws.NewSocketManager()
		id := uuid.New()

		p, err := m.GetOrCreatePool(id)
		assert.NoError(t, err)
		err = m.AppendPool(p)
		assert.Equal(t, swaggerws.ErrPoolAlreadyInContainer, err)

		pp, err := m.GetPoolById(id)
		assert.NoError(t, err)
		assert.Equal(t, p, pp)
	})

	t.Run("test Destroy() method", func(t *testing.T) {
		m := swaggerws.NewSocketManager()

		assert.Equal(t, true, m.IsReady())
		assert.Equal(t, false, m.IsDestroyed())

		p, _, _ := getNewPool(uuid.New())
		_ = m.AppendPool(p)
		p, _, _ = getNewPool(uuid.New())
		_ = m.AppendPool(p)
		p, _, _ = getNewPool(uuid.New())
		_ = m.AppendPool(p)
		assert.Equal(t, true, m.IsReady())
		assert.Equal(t, false, m.IsDestroyed())

		_ = m.Destroy()
		assert.Equal(t, false, m.IsReady())
		assert.Equal(t, true, m.IsDestroyed())

		errs := m.Destroy()
		assert.Equal(t, 1, len(errs))
		assert.Equal(t, swaggerws.ErrManagerDestroyed, <-errs)

		_, err := m.GetPoolById(uuid.New())
		assert.Equal(t, err, swaggerws.ErrManagerDestroyed)
	})
}

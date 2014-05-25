package alice

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests creating a new chain
func TestNew(t *testing.T) {
	c1 := func(h http.Handler) http.Handler {
		return nil
	}
	c2 := func(h http.Handler) http.Handler {
		return http.StripPrefix("potato", nil)
	}

	slice := []Constructor{c1, c2}

	chain := New(slice...)
	assert.Equal(t, chain.constructors[0], slice[0])
	assert.Equal(t, chain.constructors[1], slice[1])
}

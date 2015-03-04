// Package alice implements a middleware chaining solution.
package alice

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A constructor for middleware
// that writes its own "tag" into the RW and does nothing else.
// Useful in checking if a chain is behaving in the right order.
func tagMiddleware(tag string) Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(tag))
			h.ServeHTTP(w, r)
		})
	}
}

var testApp = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("app\n"))
})

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
	assert.Equal(t, chain[0], slice[0])
	assert.Equal(t, chain[1], slice[1])
}

func TestThenWorksWithNoMiddleware(t *testing.T) {
	assert.NotPanics(t, func() {
		chain := New()
		final := chain.Then(testApp)

		assert.Equal(t, final, testApp)
	})
}

func TestThenTreatsNilAsDefaultServeMux(t *testing.T) {
	chained := New().Then(nil)
	assert.Equal(t, chained, http.DefaultServeMux)
}

func TestThenFuncTreatsNilAsDefaultServeMux(t *testing.T) {
	chained := New().ThenFunc(nil)
	assert.Equal(t, chained, http.DefaultServeMux)
}

func TestThenOrdersHandlersRight(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	chained := New(t1, t2, t3).Then(testApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)

	assert.Equal(t, w.Body.String(), "t1\nt2\nt3\napp\n")
}

func TestAppendAddsHandlersCorrectly(t *testing.T) {
	chain := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	newChain := chain.Append(tagMiddleware("t3\n"), tagMiddleware("t4\n"))

	assert.Equal(t, len(chain), 2)
	assert.Equal(t, len(newChain), 4)

	chained := newChain.Then(testApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)

	assert.Equal(t, w.Body.String(), "t1\nt2\nt3\nt4\napp\n")
}

func TestAppendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware(""))
	newChain := chain.Append(tagMiddleware(""))

	assert.NotEqual(t, &chain[0], &newChain[0])
}

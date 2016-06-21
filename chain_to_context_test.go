package alice

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"golang.org/x/net/context"
)

// A constructor for middleware
// that writes its own "tag" into the RW and does nothing else.
// Useful in checking if a chain is behaving in the right order.
func ctxTagMiddleware(tag string) ContextualisedConstructor {
	return func(h ContextualisedHandler) ContextualisedHandler {
		return ContextualisedHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(tag))
			h.ServeHTTPC(ctx, w, r)
		})
	}
}

var ctxTestApp = func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("app\n"))
}

// Tests creating a new context capable chain
func TestAppendContext(t *testing.T) {
	c1 := tagMiddleware("t1\n")
	c2 := tagMiddleware("t2\n")

	slice := []Constructor{c1, c2}

	chain := New(slice...)

	assert.True(t, funcsEqual(chain.constructors[0], slice[0]))
	assert.True(t, funcsEqual(chain.constructors[1], slice[1]))

	c2c := func(next ContextualisedHandler) http.Handler {
		bg := context.Background()
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ctx\n"))
			next.ServeHTTPC(bg, w, r)
		})
	}

	toCtxchain := chain.Contextualise(c2c)

	assert.True(t, funcsEqual(toCtxchain.transformer, c2c))

	assert.True(t, len(toCtxchain.chain.constructors) != 0)

	assert.True(t, funcsEqual(toCtxchain.chain.constructors[0], slice[0]))
	assert.True(t, funcsEqual(toCtxchain.chain.constructors[1], slice[1]))

	toCtxchain = toCtxchain.Append(ctxTagMiddleware("ct1\n"), ctxTagMiddleware("ct2\n"))

	assert.True(t, funcsEqual(toCtxchain.chain.constructors[0], slice[0]))
	assert.True(t, funcsEqual(toCtxchain.chain.constructors[1], slice[1]))

	cchained := toCtxchain.ThenFunc(ctxTestApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	cchained.ServeHTTP(w, r)

	assert.Equal(t, w.Body.String(), "t1\nt2\nctx\nct1\nct2\napp\n")
}

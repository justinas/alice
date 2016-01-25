package alice

import (
	"net/http"

	"golang.org/x/net/context"
)

//ContextualizedConstructor is a Constructor with a context
type ContextualizedConstructor func(ContextualizedHandler) ContextualizedHandler

//ContextualizedHandler is a http.Handler with a context
type ContextualizedHandler interface {
	ServeHTTP(context.Context, http.ResponseWriter, *http.Request)
}

//ContextualizedHandlerFunc is a http.HandlerFunc with a context
type ContextualizedHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (f ContextualizedHandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	f(ctx, w, r)
}

type ContextualizedChain struct {
	constructors []ContextualizedConstructor
}

func NewContextualized(constructors ...ContextualizedConstructor) (cc ContextualizedChain) {
	cc.constructors = append(cc.constructors, constructors...)
	return
}

// Append extends a contextualized chain, adding the specified constructors
// as the last ones in the request flow.
//
// Append returns a new chain, leaving the original one untouched.
//
//     stdChain := alice.New(m1, m2)
//     extChain := stdChain.Append(m3, m4)
//     // requests in stdChain go m1 -> m2
//     // requests in extChain go m1 -> m2 -> m3 -> m4
func (c ContextualizedChain) Append(constructors ...ContextualizedConstructor) ContextualizedChain {
	newCons := make([]ContextualizedConstructor, len(c.constructors)+len(constructors))
	copy(newCons, c.constructors)
	copy(newCons[len(c.constructors):], constructors)

	return NewContextualized(newCons...)
}

// Then works identically to Chain.Then but with a contextualized handler
func (cc ContextualizedChain) Then(fn ContextualizedHandler) ContextualizedHandler {
	if fn == nil {
		fn = ContextualizedHandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})
	}

	for i := len(cc.constructors) - 1; i >= 0; i-- {
		fn = cc.constructors[i](fn)
	}
	return fn
}

func (cc ContextualizedChain) ThenFunc(fn ContextualizedHandlerFunc) ContextualizedHandler {
	if fn == nil {
		return cc.Then(nil)
	}
	return cc.Then(fn)
}

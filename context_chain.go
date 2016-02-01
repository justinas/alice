package alice

import (
	"net/http"

	"golang.org/x/net/context"
)

//ContextualizedConstructor is a Constructor with a context
type ContextualizedConstructor func(ContextualizedHandler) ContextualizedHandler

//ContextualizedHandler is a http.Handler with a context
type ContextualizedHandler interface {
	ServeHTTPC(context.Context, http.ResponseWriter, *http.Request)
}

//ContextualizedHandlerFunc is a http.HandlerFunc with a context
type ContextualizedHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

//ServeHTTPC is like serve http but with a context
func (f ContextualizedHandlerFunc) ServeHTTPC(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	f(ctx, w, r)
}

//ContextualizedChain is a chain of contextualised handlers
//it behaves just like Chain
type ContextualizedChain struct {
	constructors []ContextualizedConstructor
}

//NewContextualized instantiates a new Chain of contextualised http handlers
//Just like New
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
func (cc ContextualizedChain) Append(constructors ...ContextualizedConstructor) ContextualizedChain {
	newCons := make([]ContextualizedConstructor, len(cc.constructors)+len(constructors))
	copy(newCons, cc.constructors)
	copy(newCons[len(cc.constructors):], constructors)

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

// ThenFunc works identically to Chain.ThenFunc but with a contextualized handler
func (cc ContextualizedChain) ThenFunc(fn ContextualizedHandlerFunc) ContextualizedHandler {
	if fn == nil {
		return cc.Then(nil)
	}
	return cc.Then(fn)
}

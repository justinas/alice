package alice

import (
	"net/http"

	"golang.org/x/net/context"
)

//ContextualisedConstructor is a Constructor with a context
type ContextualisedConstructor func(ContextualisedHandler) ContextualisedHandler

//ContextualisedHandler is a http.Handler with a context
type ContextualisedHandler interface {
	ServeHTTPC(context.Context, http.ResponseWriter, *http.Request)
}

//ContextualisedHandlerFunc is a http.HandlerFunc with a context
type ContextualisedHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

//ServeHTTPC is like serve http but with a context
func (f ContextualisedHandlerFunc) ServeHTTPC(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	f(ctx, w, r)
}

//ContextualisedChain is a chain of contextualised handlers
//it behaves just like Chain
type ContextualisedChain struct {
	constructors []ContextualisedConstructor
}

//NewContextualised instantiates a new Chain of contextualised http handlers
//Just like New
func NewContextualised(constructors ...ContextualisedConstructor) (cc ContextualisedChain) {
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
func (cc ContextualisedChain) Append(constructors ...ContextualisedConstructor) ContextualisedChain {
	newCons := make([]ContextualisedConstructor, len(cc.constructors)+len(constructors))
	copy(newCons, cc.constructors)
	copy(newCons[len(cc.constructors):], constructors)

	return NewContextualised(newCons...)
}

// Then works identically to Chain.Then but with a contextualized handler
func (cc ContextualisedChain) Then(fn ContextualisedHandler) ContextualisedHandler {
	if fn == nil {
		fn = ContextualisedHandlerFunc(func(_ context.Context, w http.ResponseWriter, r *http.Request) {
			http.DefaultServeMux.ServeHTTP(w, r)
		})
	}

	for i := len(cc.constructors) - 1; i >= 0; i-- {
		fn = cc.constructors[i](fn)
	}
	return fn
}

// ThenFunc works identically to Chain.ThenFunc but with a contextualized handler
func (cc ContextualisedChain) ThenFunc(fn ContextualisedHandlerFunc) ContextualisedHandler {
	if fn == nil {
		return cc.Then(nil)
	}
	return cc.Then(fn)
}

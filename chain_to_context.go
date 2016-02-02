package alice

import "net/http"

//ToContextConstructor allows you to chain a non contextualized
//http handler with a contextualized one.
type ToContextConstructor func(ContextualisedHandler) http.Handler

// Contextualise allows you to append a contextualized http handler
// to your normal chain thus allowing to add ctx support to all
// subsequent http handlers.
func (c Chain) Contextualise(transformer ToContextConstructor) (cc toContextualisedChain) {
	return toContextualisedChain{
		chain:       c.copy(),
		transformer: transformer,
	}
}

// toContextualisedChain acts as a list of non contextualized http.Handlers and then contextualized http.Handlers.
// toContextualisedChain is effectively immutable:
// once created, it will always hold
// the same set of constructors in the same order.
type toContextualisedChain struct {
	chain       Chain
	transformer ToContextConstructor
	cchain      ContextualisedChain
}

func (tcc toContextualisedChain) Append(constructors ...ContextualisedConstructor) toContextualisedChain {
	return toContextualisedChain{
		chain:       tcc.chain.copy(),
		transformer: tcc.transformer,
		cchain:      tcc.cchain.Append(constructors...),
	}
}

// Then works identically to Chain.Then but with a contextualized handler
func (tcc toContextualisedChain) Then(cfinal ContextualisedHandler) http.Handler {
	cfinal = tcc.cchain.Then(cfinal)

	final := tcc.transformer(cfinal)

	return tcc.chain.Then(final)
}

func (tcc toContextualisedChain) ThenFunc(fn ContextualisedHandlerFunc) http.Handler {
	if fn == nil {
		return tcc.Then(nil)
	}
	return tcc.Then(fn)
}

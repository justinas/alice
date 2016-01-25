package alice

import "net/http"

//ToContextConstructor allows you to chain a non contextualized
//http handler with a contextualized one.
type ToContextConstructor func(ContextualizedHandler) http.Handler

// Contextualize allows you to append a contextualized http handler
// to your normal chain thus allowing to add ctx support to all
// subsequent http handlers.
func (c Chain) Contextualize(transformer ToContextConstructor) (cc toContextualizedChain) {
	return toContextualizedChain{
		chain:       c.copy(),
		transformer: transformer,
	}
}

// toContextualizedChain acts as a list of non contextualized http.Handlers and then contextualized http.Handlers.
// toContextualizedChain is effectively immutable:
// once created, it will always hold
// the same set of constructors in the same order.
type toContextualizedChain struct {
	chain       Chain
	transformer ToContextConstructor
	cchain      ContextualizedChain
}

func (tcc toContextualizedChain) Append(constructors ...ContextualizedConstructor) toContextualizedChain {
	return toContextualizedChain{
		chain:       tcc.chain.copy(),
		transformer: tcc.transformer,
		cchain:      tcc.cchain.Append(constructors...),
	}
}

// Then works identically to Chain.Then but with a contextualized handler
func (tcc toContextualizedChain) Then(cfinal ContextualizedHandler) http.Handler {
	cfinal = tcc.cchain.Then(cfinal)

	final := tcc.transformer(cfinal)

	return tcc.chain.Then(final)
}

func (tcc toContextualizedChain) ThenFunc(fn ContextualizedHandlerFunc) http.Handler {
	if fn == nil {
		return tcc.Then(nil)
	}
	return tcc.Then(fn)
}

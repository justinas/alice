// Package alice provides a convenient way to chain http handlers.
package alice

import "net/http"

// Chain acts as a list of http.Handler handlers.
// Chain is effectively immutable:
// once created, it will always hold
// the same set of handlers in the same order.
type Chain struct {
	handlers []func(http.Handler) http.Handler
}

// New creates a new chain,
// memorizing the given list of middleware handlers.
// New serves no other function,
// http.handlers are only called upon a call to Then().
func New(handlers ...func(http.Handler) http.Handler) Chain {
	c := Chain{}
	c.handlers = append(c.handlers, handlers...)

	return c
}

// Then chains the middleware and returns the final http.Handler.
//     New(m1, m2, m3).Then(h)
// is equivalent to:
//     m1(m2(m3(h)))
// When the request comes in, it will be passed to m1, then m2, then m3
// and finally, the given handler
// (assuming every middleware calls the following one).
//
// A chain can be safely reused by calling Then() several times.
//     stdStack := alice.New(ratelimitHandler, csrfHandler)
//     indexPipe = stdStack.Then(indexHandler)
//     authPipe = stdStack.Then(authHandler)
// Note that handlers are called on every call to Then()
// and thus several instances of the same middleware will be created
// when a chain is reused in this way.
// For proper middleware, this should cause no problems.
//
// Then() treats nil as http.DefaultServeMux.
func (c Chain) Then(h http.Handler) http.Handler {
	var final http.Handler
	if h != nil {
		final = h
	} else {
		final = http.DefaultServeMux
	}

	for i := len(c.handlers) - 1; i >= 0; i-- {
		final = c.handlers[i](final)
	}

	return final
}

// ThenFunc works identically to Then, but takes
// a HandlerFunc instead of a Handler.
//
// The following two statements are equivalent:
//     c.Then(http.HandlerFunc(fn))
//     c.ThenFunc(fn)
//
// ThenFunc provides all the guarantees of Then.
func (c Chain) ThenFunc(fn http.HandlerFunc) http.Handler {
	if fn == nil {
		return c.Then(nil)
	}
	return c.Then(http.HandlerFunc(fn))
}

// Append extends a chain, adding the specified handlers
// as the last ones in the request flow.
//
// Append returns a new chain, leaving the original one untouched.
//
//     stdChain := alice.New(m1, m2)
//     extChain := stdChain.Append(m3, m4)
//     // requests in stdChain go m1 -> m2
//     // requests in extChain go m1 -> m2 -> m3 -> m4
func (c Chain) Append(handlers ...func(http.Handler) http.Handler) Chain {
	newCons := make([]func(http.Handler) http.Handler, len(c.handlers)+len(handlers))
	copy(newCons, c.handlers)
	copy(newCons[len(c.handlers):], handlers)

	newChain := New(newCons...)
	return newChain
}

// Extend extends a chain by adding the specified chain
// as the last one in the request flow.
//
// Extend returns a new chain, leaving the original one untouched.
//
//     stdChain := alice.New(m1, m2)
//     ext1Chain := alice.New(m3, m4)
//     ext2Chain := stdChain.Extend(ext1Chain)
//     // requests in stdChain go  m1 -> m2
//     // requests in ext1Chain go m3 -> m4
//     // requests in ext2Chain go m1 -> m2 -> m3 -> m4
//
// Another example:
//  aHtmlAfterNosurf := alice.New(m2)
// 	aHtml := alice.New(m1, func(h http.Handler) http.Handler {
// 		csrf := nosurf.New(h)
// 		csrf.SetFailureHandler(aHtmlAfterNosurf.ThenFunc(csrfFail))
// 		return csrf
// 	}).Extend(aHtmlAfterNosurf)
//		// requests to aHtml hitting nosurfs success handler go m1 -> nosurf -> m2 -> target-handler
//		// requests to aHtml hitting nosurfs failure handler go m1 -> nosurf -> m2 -> csrfFail
func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain.handlers...)
}

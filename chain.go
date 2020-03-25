// Package alice provides a convenient way to chain http handlers.
package alice

import (
	"net/http"
)

// A constructor for a piece of middleware.
// Some middleware use this constructor out of the box,
// so in most cases you can just pass somepackage.New
type Constructor func(http.Handler) http.Handler

// Chain acts as a list of http.Handler constructors.
// Chain is effectively immutable:
// once created, it will always hold
// the same set of constructors in the same order.
type Chain struct {
	constructors []Constructor
	endwares     []Endware
}

// New creates a new chain,
// memorizing the given list of middleware constructors.
// New serves no other function,
// constructors are only called upon a call to Then().
func New(constructors ...Constructor) Chain {
	return Chain{append(([]Constructor)(nil), constructors...), ([]Endware)(nil)}
}

// endwareHandler represents a handler that has been modified
// to execute endwares afterwards. This is a helper for Then()
// because if we just wrap it in an anonymous
// 		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request)))
// there is a stack overflow
type endwareHandler struct {
	handler  http.Handler
	endwares []Endware
}

// ServeHTTP serves the main endwareHandler's handler as well as
// calling all of the individual endwares afterwards.
func (eh endwareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eh.handler.ServeHTTP(w, r)
	for _, endware := range eh.endwares {
		endware.ServeHTTP(w, r)
	}
}

// Then chains the middleware and endwares and returns the final http.Handler.
//     New(m1, m2, m3).After(e1, e2, e3).Then(h)
// is equivalent to:
//     m1(m2(m3(h)))
// followed by:
//     e1(e2(e3()))
// When the request comes in, it will be passed to m1, then m2, then m3,
// then the given handler (who serves the response), then e1, e2, e3
// (assuming every middleware/endwares calls the following one).
//
// A chain can be safely reused by calling Then() several times.
//     stdStack := alice.New(ratelimitHandler, csrfHandler).After(loggingHandler)
//     indexPipe = stdStack.Then(indexHandler)
//     authPipe = stdStack.Then(authHandler)
// Note that constructors and endwares are called on every call to Then()
// and thus several instances of the same middleware/endwares will be created
// when a chain is reused in this way.
// For proper middleware/endwares, this should cause no problems.
//
// Then() treats nil as http.DefaultServeMux.
func (c Chain) Then(h http.Handler) http.Handler {
	if h == nil {
		h = http.DefaultServeMux
	}

	if len(c.endwares) > 0 {
		h = endwareHandler{h, c.endwares}
	}

	for i := range c.constructors {
		h = c.constructors[len(c.constructors)-1-i](h)
	}

	return h
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
	return c.Then(fn)
}

// Append extends a chain, adding the specified constructors
// as the last ones in the request flow.
//
// Append returns a new chain, leaving the original one untouched.
// The new chain will have the original chain's endwares.
//
//     stdChain := alice.New(m1, m2)
//     extChain := stdChain.Append(m3, m4)
//     // requests in stdChain go m1 -> m2
//     // requests in extChain go m1 -> m2 -> m3 -> m4
func (c Chain) Append(constructors ...Constructor) Chain {
	newCons := make([]Constructor, 0, len(c.constructors)+len(constructors))
	newCons = append(newCons, c.constructors...)
	newCons = append(newCons, constructors...)

	return New(newCons...).AppendEndware(c.endwares...)
}

// Extend extends a chain by adding the specified chain
// as the last one in the request flow.
//
// Extend returns a new chain, leaving the original one untouched.
//
//     stdChain := alice.New(m1, m2)
//     ext1Chain := alice.New(m3, m4).After(e1, e2)
//     ext2Chain := stdChain.Extend(ext1Chain)
//     // requests in stdChain  go m1 -> m2 -> handler
//     // requests in ext1Chain go m3 -> m4 -> handler -> e1 -> e2
//     // requests in ext2Chain go m1 -> m2 -> m3 -> m4 -> handler -> e1 -> e2
//
// Another example:
//  aHtmlAfterNosurf := alice.New(m2)
//  logRequestChain := aHtmlAfterNosurf.After(e1)
// 	aHtml := alice.New(m1, func(h http.Handler) http.Handler {
// 		csrf := nosurf.New(h)
// 		csrf.SetFailureHandler(logRequestChain.ThenFunc(csrfFail))
// 		return csrf
// 	}).Extend(logRequestChain)
//		// requests to aHtml hitting nosurfs success handler go:
//				m1 -> nosurf -> m2 -> target-handler -> e1
//		// requests to aHtml hitting nosurfs failure handler go:
//				m1 -> nosurf -> m2 -> csrfFail -> e1
func (c Chain) Extend(chain Chain) Chain {
	return c.
		Append(chain.constructors...).
		AppendEndware(chain.endwares...)
}

// Endware is functionality executed after a the main handler is called
// and response has been sent to the requester.  Like middleware,
// values from the request or response can be accessed. This will not
// let you access values from the request or the response that can no longer be used.
// e.g. re-reading a request body, re-setting the response headers, etc.
type Endware http.Handler

// After creates a new chain with the original chain's
// constructors and endwares, as well as the provided endwares.
// Endwares are executed after both the constructors and
// the Then() handler are called.
func (c Chain) After(endwares ...Endware) Chain {
	newEnds := make([]Endware, 0, len(c.endwares)+len(endwares))
	newEnds = append(newEnds, c.endwares...)
	newEnds = append(newEnds, endwares...)

	newC := New(c.constructors...)
	newC.endwares = newEnds
	return newC
}

// AfterFuncs works identically to After, but takes HandlerFuncs
// instead of Endwares.
//
// The following two statements are equivalent:
//     c.After(http.HandlerFunc(fn1), http.HandlerFunc(fn2))
//     c.AfterFuncs(fn1, fn2)
//
// AfterFuncs provides all the guarantees of After.
func (c Chain) AfterFuncs(fns ...func(w http.ResponseWriter, r *http.Request)) Chain {
	// convert each http.HandlerFunc into an Endware
	endwares := make([]Endware, len(fns))
	for i, fn := range fns {
		endwares[i] = http.HandlerFunc(fn)
	}

	return c.After(endwares...)
}

// AppendEndware extends a chain, adding the specified endwares
// as the last ones in the request flow.
//
// AppendEndware returns a new chain, leaving the original one untouched.
// The new chain will have the original chain's constructors.
//
//     stdChain := alice.New(m1).After(e1, e2)
//     extChain := stdChain.AppendEndware(e3, e4)
//     // requests in stdHandler go m1 -> handler -> e1 -> e2
//     // requests in extHandler go m1 -> handler -> e1 -> e2 -> e3 -> e4
func (c Chain) AppendEndware(endwares ...Endware) Chain {
	return New(c.constructors...).After(append(c.endwares, endwares...)...)
}

// AppendEndwareFuncs works identically to AppendEndware, but takes HandlerFuncs
// instead of Endwares.
//
// The following two statements are equivalent:
//     c.AppendEndware(http.HandlerFunc(fn1), http.HandlerFunc(fn2))
//     c.AppendEndwareFuncs(fn1, fn2)
//
// AppendEndwareFuncs provides all the guarantees of AppendEndware.
func (c Chain) AppendEndwareFuncs(fns ...func(w http.ResponseWriter, r *http.Request)) Chain {
	// convert each http.HandlerFunc into an Endware
	endwares := make([]Endware, len(fns))
	for i, fn := range fns {
		endwares[i] = http.HandlerFunc(fn)
	}

	return c.AppendEndware(endwares...)

}

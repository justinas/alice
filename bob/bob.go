// Package bob provides a convenient way to chain http round trippers.
package bob

import "net/http"

// RoundTripperFunc is to RoundTripper what HandlerFunc is to Handler.
// It is a higher-order function that enables chaining of RoundTrippers
// with the middleware pattern.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls the function itself.
func (f RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// Constructor for a piece of middleware.
// Some middleware uses this constructor out of the box,
// so in most cases you can just pass somepackage.New
type Constructor func(http.RoundTripper) http.RoundTripper

// Chain acts as a list of http.RoundTripper constructors.
// Chain is effectively immutable:
// once created, it will always hold
// the same set of constructors in the same order.
type Chain struct {
	constructors []Constructor
}

// New creates a new chain,
// memorizing the given list of middleware constructors.
// New serves no other function,
// constructors are only called upon a call to Then().
func New(constructors ...Constructor) Chain {
	return Chain{append(([]Constructor)(nil), constructors...)}
}

// Then chains the middleware and returns the final http.RoundTripper.
//     New(m1, m2, m3).Then(rt)
// is equivalent to:
//     m1(m2(m3(rt)))
// When the request goes out, it will be passed to m1, then m2, then m3
// and finally, the given round tripper
// (assuming every middleware calls the following one).
//
// A chain can be safely reused by calling Then() several times.
//     stdStack := chain.New(ratelimitHandler, csrfHandler)
//     indexPipe = stdStack.Then(indexHandler)
//     authPipe = stdStack.Then(authHandler)
// Note that constructors are called on every call to Then()
// and thus several instances of the same middleware will be created
// when a chain is reused in this way.
// For proper middleware, this should cause no problems.
//
// Then() treats nil as http.DefaultTransport.
func (c Chain) Then(rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}

	for i := range c.constructors {
		rt = c.constructors[len(c.constructors)-1-i](rt)
	}

	return rt
}

// ThenFunc works identically to Then, but takes
// a RoundTripperFunc instead of a RoundTripper.
//
// The following two statements are equivalent:
//     c.Then(http.RoundTripperFunc(fn))
//     c.ThenFunc(fn)
//
// RoundTripperFunc provides all the guarantees of Then.
func (c Chain) ThenFunc(fn RoundTripperFunc) http.RoundTripper {
	if fn == nil {
		return c.Then(nil)
	}
	return c.Then(fn)
}

// Append extends a chain, adding the specified constructors
// as the last ones in the request flow.
//
// Append returns a new chain, leaving the original one untouched.
//
//     stdChain := chain.New(m1, m2)
//     extChain := stdChain.Append(m3, m4)
//     // requests in stdChain go m1 -> m2
//     // requests in extChain go m1 -> m2 -> m3 -> m4
func (c Chain) Append(constructors ...Constructor) Chain {
	newCons := make([]Constructor, 0, len(c.constructors)+len(constructors))
	newCons = append(newCons, c.constructors...)
	newCons = append(newCons, constructors...)

	return Chain{newCons}
}

// Extend extends a chain by adding the specified chain
// as the last one in the request flow.
//
// Extend returns a new chain, leaving the original one untouched.
//
//     stdChain := chain.New(m1, m2)
//     ext1Chain := chain.New(m3, m4)
//     ext2Chain := stdChain.Extend(ext1Chain)
//     // requests in stdChain go  m1 -> m2
//     // requests in ext1Chain go m3 -> m4
//     // requests in ext2Chain go m1 -> m2 -> m3 -> m4
//
// Another example:
//  aHtmlAfterNosurf := chain.New(m2)
// 	aHtml := chain.New(m1, func(rt http.RoundTripper) http.RoundTripper {
// 		csrf := nosurf.New(rt)
// 		csrf.SetFailureHandler(aHtmlAfterNosurf.ThenFunc(csrfFail))
// 		return csrf
// 	}).Extend(aHtmlAfterNosurf)
//		// requests to aHtml hitting nosurfs success handler go m1 -> nosurf -> m2 -> target-roundtripper
//		// requests to aHtml hitting nosurfs failure handler go m1 -> nosurf -> m2 -> csrfFail
func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain.constructors...)
}

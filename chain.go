package alice

import "net/http"

// A constructor for a piece of middleware.
// Most middleware use this constructor out of the box,
// so in most cases you can just pass somepackage.New
type Constructor func(http.Handler) http.Handler

type Chain struct {
	constructors []Constructor
}

// Creates a new chain, memorizing the given middleware constructors
func New(constructors ...Constructor) Chain {
	c := Chain{}
	c.constructors = append(c.constructors, constructors...)

	return c
}

// Chains the middleware and returns the final http.Handler
//     New(m1, m2, m3).Then(h)
// is equivalent to:
//     m1(m2(m3(h)))
// When the request comes in, it will be passed to m1, then m2, then m3
// and finally, the given handler
// (assuming every middleware calls the following one)
func (c Chain) Then(h http.Handler) http.Handler {
	var final http.Handler
	if h != nil {
		final = h
	} else {
		final = http.DefaultServeMux
	}

	for i := len(c.constructors) - 1; i >= 0; i-- {
		final = c.constructors[i](final)
	}

	return final
}

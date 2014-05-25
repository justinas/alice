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

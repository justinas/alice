// Package alice implements a middleware chaining solution.
package alice

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// A constructor for middleware
// that writes its own "tag" into the RW and does nothing else.
// Useful in checking if a chain is behaving in the right order.
func tagMiddleware(tag string) Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(tag))
			h.ServeHTTP(w, r)
		})
	}
}

func tagEndware(tag string) Endware {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(tag))
	})
}

// Not recommended (https://golang.org/pkg/reflect/#Value.Pointer),
// but the best we can do.
func funcsEqual(f1, f2 interface{}) bool {
	val1 := reflect.ValueOf(f1)
	val2 := reflect.ValueOf(f2)
	return val1.Pointer() == val2.Pointer()
}

var testApp = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("app\n"))
})

func TestNew(t *testing.T) {
	c1 := func(h http.Handler) http.Handler {
		return nil
	}

	c2 := func(h http.Handler) http.Handler {
		return http.StripPrefix("potato", nil)
	}

	slice := []Constructor{c1, c2}

	chain := New(slice...)
	for k := range slice {
		if !funcsEqual(chain.constructors[k], slice[k]) {
			t.Error("New does not add constructors correctly")
		}
	}
}

func TestAfter(t *testing.T) {
	e1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	e2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	slice := []Endware{e1, e2}

	chain := New().After(slice...)
	for k := range slice {
		if !funcsEqual(chain.endwares[k], slice[k]) {
			t.Error("After does not add endwares correctly")
		}
	}
}

func TestThenWorksWithNoMiddleware(t *testing.T) {
	if !funcsEqual(New().Then(testApp), testApp) {
		t.Error("Then does not work with no middleware")
	}
}

func TestThenWorksWithNoEndware(t *testing.T) {
	if !funcsEqual(New().After().Then(testApp), testApp) {
		t.Error("Then does not work with no endware")
	}
}

func TestThenTreatsNilAsDefaultServeMux(t *testing.T) {
	if New().Then(nil) != http.DefaultServeMux {
		t.Error("Then does not treat nil as DefaultServeMux")
	}
}

func TestThenFuncTreatsNilAsDefaultServeMux(t *testing.T) {
	if New().ThenFunc(nil) != http.DefaultServeMux {
		t.Error("ThenFunc does not treat nil as DefaultServeMux")
	}
}

func TestThenFuncConstructsHandlerFunc(t *testing.T) {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	chained := New().ThenFunc(fn)
	rec := httptest.NewRecorder()

	chained.ServeHTTP(rec, (*http.Request)(nil))

	if reflect.TypeOf(chained) != reflect.TypeOf((http.HandlerFunc)(nil)) {
		t.Error("ThenFunc does not construct HandlerFunc")
	}
}

func TestThenOrdersHandlersCorrectly(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")
	e1 := tagEndware("e1\n")
	e2 := tagEndware("e2\n")
	e3 := tagEndware("e3\n")

	chained := New(t1, t2, t3).After(e1, e2, e3).Then(testApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)

	if w.Body.String() != "t1\nt2\nt3\napp\ne1\ne2\ne3\n" {
		t.Error("Then does not order handlers correctly")
	}
}

func TestAppendAddsHandlersCorrectly(t *testing.T) {
	chain := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	newChain := chain.Append(tagMiddleware("t3\n"), tagMiddleware("t4\n"))

	if len(chain.constructors) != 2 {
		t.Error("chain should have 2 constructors")
	}
	if len(chain.endwares) != 0 {
		t.Error("chain should have 0 endwares")
	}
	if len(newChain.constructors) != 4 {
		t.Error("newChain should have 4 constructors")
	}
	if len(newChain.endwares) != 0 {
		t.Error("newChain should have 0 endwares")
	}

	chained := newChain.Then(testApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)

	if w.Body.String() != "t1\nt2\nt3\nt4\napp\n" {
		t.Error("Append does not add handlers correctly")
	}
}

func TestAppendEndwareAddsHandlersCorrectly(t *testing.T) {
	chain := New(tagMiddleware("t1\n")).After(tagEndware("e1\n"), tagEndware("e2\n"))
	newChain := chain.AppendEndware(tagEndware("e3\n"), tagEndware("e4\n"))

	if len(chain.constructors) != 1 {
		t.Error("chain should have 1 constructor")
	}
	if len(chain.endwares) != 2 {
		t.Error("chain should have 2 endwares")
	}
	if len(newChain.constructors) != 1 {
		t.Error("newChain should have 1 constructor")
	}
	if len(newChain.endwares) != 4 {
		t.Error("newChain should have 4 endwares")
	}

	chained := newChain.Then(testApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)

	if w.Body.String() != "t1\napp\ne1\ne2\ne3\ne4\n" {
		t.Error("AppendEndware does not add handlers correctly")
	}
}

func TestAppendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware("")).After(tagEndware(""))
	newChain := chain.Append(tagMiddleware(""))

	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("Append does not respect constructor immutability")
	}

	if &chain.endwares[0] == &newChain.endwares[0] {
		t.Error("Append does not respect endware immutability")
	}
}

func TestAppendEndwareRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware("")).After(tagEndware(""))
	newChain := chain.AppendEndware(tagEndware(""))

	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("AppendEndware does not respect constructor immutability")
	}

	if &chain.endwares[0] == &newChain.endwares[0] {
		t.Error("AppendEndware does not respect endware immutability")
	}
}

func TestExtendsRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware("")).After(tagEndware(""))
	newChain := chain.Extend(New(tagMiddleware("")))

	// chain.constructors[0] should have the same functionality as
	// newChain.constructors[1], but check both anyways
	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("Extends does not respect constructor immutability")
	}

	if &chain.constructors[0] == &newChain.constructors[1] {
		t.Error("Extends does not respect constructor immutability")
	}

	if &chain.endwares[0] == &newChain.endwares[0] {
		t.Error("Extends does not respect endware immutability")
	}
}

func TestExtendAddsHandlersCorrectly(t *testing.T) {
	chain1 := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	chain2 := New(tagMiddleware("t3\n"), tagMiddleware("t4\n")).
		After(tagEndware("e1\n"), tagEndware("e2\n"))
	newChain := chain1.Extend(chain2)

	if len(chain1.constructors) != 2 {
		t.Error("chain1 should contain 2 constructors")
	}
	if len(chain1.endwares) != 0 {
		t.Error("chain1 should contain 0 endwares")
	}

	if len(chain2.constructors) != 2 {
		t.Error("chain2 should contain 2 constructors")
	}
	if len(chain2.endwares) != 2 {
		t.Error("chain2 should contain 2 endwares")
	}

	if len(newChain.constructors) != 4 {
		t.Error("newChain should contain 4 constructors")
	}
	if len(newChain.endwares) != 2 {
		t.Error("newChain should contain 2 endwares")
	}

	chained := newChain.Then(testApp)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)

	if w.Body.String() != "t1\nt2\nt3\nt4\napp\ne1\ne2\n" {
		t.Error("Extend does not add handlers in correctly")
	}
}

func TestExtendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware("")).After(tagEndware(""))
	newChain := chain.Extend(New(tagMiddleware("")))

	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("Extend does not respect immutability")
	}

	if &chain.endwares[0] == &newChain.endwares[0] {
		t.Error("Extend does not respect immutability for endwares")
	}
}

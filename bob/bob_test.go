// Package bob provides a convenient way to chain http round trippers.
package bob

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
)

// A constructor for middleware
// that writes its own "tag" into the request body and does nothing else.
// Useful in checking if a chain is behaving in the right order.
func tagMiddleware(tag string) Constructor {
	return func(rt http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			err := appendTag(tag, r)
			if err != nil {
				return nil, err
			}
			return rt.RoundTrip(r)
		})
	}
}

func appendTag(tag string, r *http.Request) error {
	var newBody []byte
	if r.Body == nil {
		newBody = []byte(tag)
	} else {
		body, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return err
		}
		newBody = append(body, []byte(tag)...)
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(newBody))
	return nil
}

// Not recommended (https://golang.org/pkg/reflect/#Value.Pointer),
// but the best we can do.
func funcsEqual(f1, f2 interface{}) bool {
	val1 := reflect.ValueOf(f1)
	val2 := reflect.ValueOf(f2)
	return val1.Pointer() == val2.Pointer()
}

var testApp = RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
	appendTag("app\n", r)
	return &http.Response{}, nil
})

func TestNew(t *testing.T) {
	c1 := func(h http.RoundTripper) http.RoundTripper {
		return nil
	}

	c2 := func(h http.RoundTripper) http.RoundTripper {
		return http.DefaultTransport
	}

	slice := []Constructor{c1, c2}

	chain := New(slice...)
	for k := range slice {
		if !funcsEqual(chain.constructors[k], slice[k]) {
			t.Error("New does not add constructors correctly")
		}
	}
}

func TestThenWorksWithNoMiddleware(t *testing.T) {
	if !funcsEqual(New().Then(testApp), testApp) {
		t.Error("Then does not work with no middleware")
	}
}

func TestThenTreatsNilAsDefaultTransport(t *testing.T) {
	if New().Then(nil) != http.DefaultTransport {
		t.Error("Then does not treat nil as DefaultTransport")
	}
}

func TestThenFuncTreatsNilAsDefaultTransport(t *testing.T) {
	if New().ThenFunc(nil) != http.DefaultTransport {
		t.Error("ThenFunc does not treat nil as DefaultTransport")
	}
}

func TestThenFuncConstructsRoundTripperFunc(t *testing.T) {
	fn := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{}, nil
	})
	chained := New().ThenFunc(fn)

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.RoundTrip(r)

	if reflect.TypeOf(chained) != reflect.TypeOf((RoundTripperFunc)(nil)) {
		t.Error("ThenFunc does not construct RoundTripperFunc")
	}
}

func bodyAsString(r *http.Request) (string, error) {
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return "", err
	}
	return string(body[:]), nil
}

func TestThenOrdersRoundTrippersCorrectly(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	chained := New(t1, t2, t3).Then(testApp)

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.RoundTrip(r)

	body, err := bodyAsString(r)
	if err != nil {
		t.Fatal(err)
	}
	if body != "t1\nt2\nt3\napp\n" {
		t.Error("Then does not order round trippers correctly")
	}
}

func TestAppendAddsRoundTrippersCorrectly(t *testing.T) {
	chain := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	newChain := chain.Append(tagMiddleware("t3\n"), tagMiddleware("t4\n"))

	if len(chain.constructors) != 2 {
		t.Error("chain should have 2 constructors")
	}
	if len(newChain.constructors) != 4 {
		t.Error("newChain should have 4 constructors")
	}

	chained := newChain.Then(testApp)

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.RoundTrip(r)

	body, err := bodyAsString(r)
	if err != nil {
		t.Fatal(err)
	}
	if body != "t1\nt2\nt3\nt4\napp\n" {
		t.Error("Append does not add round trippers correctly")
	}
}

func TestAppendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware(""))
	newChain := chain.Append(tagMiddleware(""))

	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("Apppend does not respect immutability")
	}
}

func TestExtendAddsRoundTrippersCorrectly(t *testing.T) {
	chain1 := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	chain2 := New(tagMiddleware("t3\n"), tagMiddleware("t4\n"))
	newChain := chain1.Extend(chain2)

	if len(chain1.constructors) != 2 {
		t.Error("chain1 should contain 2 constructors")
	}
	if len(chain2.constructors) != 2 {
		t.Error("chain2 should contain 2 constructors")
	}
	if len(newChain.constructors) != 4 {
		t.Error("newChain should contain 4 constructors")
	}

	chained := newChain.Then(testApp)

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.RoundTrip(r)

	body, err := bodyAsString(r)
	if err != nil {
		t.Fatal(err)
	}
	if body != "t1\nt2\nt3\nt4\napp\n" {
		t.Error("Extend does not add round trippers in correctly")
	}
}

func TestExtendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware(""))
	newChain := chain.Extend(New(tagMiddleware("")))

	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("Extend does not respect immutability")
	}
}

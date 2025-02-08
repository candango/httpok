package testrunner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// HttpTestRunner is a runner that facilitates testing of HTTP requests.
//
// It allows for configuring various aspects of the request, such as clearing
// the body, header, header function, and values.
type HttpTestRunner struct {
	// clearBody indicates whether the body should be cleared after a test run.
	clearBody bool

	// clearHeader indicates whether the header should be cleared after a test
	// run.
	clearHeader bool

	// clearHandlerFunc indicates whether the handler function should be
	// cleared after a test run.
	clearHandlerFunc bool

	// clearValues indicates whether the header function should be cleared
	// after a test run.
	clearValues bool

	// body represents the body of the HTTP request.
	body io.Reader

	// header represents the header of the HTTP request.
	header http.Header

	// handler represents the HTTP handler to be tested.
	handler http.Handler

	// handlerFunc represents the HTTP handler function to be tested.
	handlerFunc func(http.ResponseWriter, *http.Request)

	// method represents the HTTP method to be tested (e.g., "GET", "POST",
	// etc.).
	method string

	// path represents the path to be tested.
	path string

	// t represents the testing instance.
	t *testing.T

	// values represents the URL values to be tested.
	values url.Values
}

// NewHttpTestRunner creates a new HttpTestRunner, equipped with empty headers
// and default HTTP method as GET.
//
// The root path is also set to its default value '/'.
//
// It's designed to streamline HTTP testing with ease and efficiency.
func NewHttpTestRunner(t *testing.T) *HttpTestRunner {
	r := &HttpTestRunner{}
	r.header = http.Header{}
	r.method = http.MethodGet
	r.path = "/"
	r.t = t
	r.values = url.Values{}
	return r
}

// Clear resets the HttpTestRunner by setting the body and handler func to nil
// while clearing the header.
//
// It also resets the clearBody, clearHandlerFunc, and clearHeader flags to
// false, ensuring a clean slate for future tests.
func (r *HttpTestRunner) Clear() *HttpTestRunner {
	r.body = nil
	r.clearBody = false
	r.handlerFunc = nil
	r.clearHandlerFunc = false
	r.header = http.Header{}
	r.clearHeader = false
	r.values = url.Values{}
	r.clearValues = false
	return r
}

// ClearBodyAfter ensures that body will be cleared after the
// HttpTestRunner.Run execution.
func (r *HttpTestRunner) ClearBodyAfter() *HttpTestRunner {
	r.clearBody = true
	return r
}

// ClearHandlerFuncAfter ensures that the handler function to be tested will be
// cleared after the HttpTestRunner.Run execution.
func (r *HttpTestRunner) ClearHandlerFuncAfter() *HttpTestRunner {
	r.clearHandlerFunc = true
	return r
}

// ClearHeaderAfter ensures that headers will be cleared after the
// HttpTestRunner.Run execution.
func (r *HttpTestRunner) ClearHeaderAfter() *HttpTestRunner {
	r.clearHeader = true
	return r
}

// WithHandlerFunc set a function to be exectued by the runner.
//
// If a function is defined, it will bypass the handler.
//
// Use ClearFuncAfter to run the function once and clear it for the next
// HttpTestRunner.Run execution.
func (r *HttpTestRunner) WithHandlerFunc(
	handlerFunc func(http.ResponseWriter, *http.Request)) *HttpTestRunner {
	r.handlerFunc = handlerFunc
	return r
}

// WithHandler set a handler to be executed by the runner.
//
// If a function is defined, it will bypass this handler.
//
// Use ClearFuncAfter to run the function once and clear it for the next
// HttpTestRunner.Run execution.
func (r *HttpTestRunner) WithHandler(handler http.Handler) *HttpTestRunner {
	r.handler = handler
	return r
}

// WithHeader add a key/value pair to be added to the header
func (r *HttpTestRunner) WithHeader(key string, value string) *HttpTestRunner {
	r.header.Add(key, value)
	return r
}

// WithPath set the path to be executed by the runner
func (r *HttpTestRunner) WithPath(path string) *HttpTestRunner {
	r.path = path
	return r
}

// WithBody set HttpTestRunner.body using an io.Reader
func (r *HttpTestRunner) WithBody(body io.Reader) *HttpTestRunner {
	r.body = body
	return r
}

// WithJsonBody set HttpTestRunner.body using an interface
func (r *HttpTestRunner) WithJsonBody(typedBody any) *HttpTestRunner {
	marshaledTypedRequest, _ := json.Marshal(typedBody)
	r.WithBody(bytes.NewReader(marshaledTypedRequest))
	return r
}

// WithStringBody set HttpTestRunner.body using a string
func (r *HttpTestRunner) WithStringBody(stringBody string) *HttpTestRunner {
	r.WithBody(bytes.NewReader([]byte(stringBody)))
	return r
}

// WithMethod set the method to be used by the runner
func (r *HttpTestRunner) WithMethod(method string) *HttpTestRunner {
	r.method = strings.ToUpper(method)
	return r
}

// WithValues set the url values to be used by the runner
func (r *HttpTestRunner) WithValues(values url.Values) *HttpTestRunner {
	r.values = values
	return r
}

// runMethod executes the HTTP request method specified by the HttpTestRunner
// struct.
//
// If runnner is running  with a WithHandlerFunc, it will bypass a defined
// handler.
func (r *HttpTestRunner) runMethod() (*http.Response, error) {
	handler := r.handler
	if r.handlerFunc != nil {
		handler = http.HandlerFunc(r.handlerFunc)
	}
	s := httptest.NewServer(handler)
	defer s.Close()
	path := r.path
	if len(r.values) > 0 {
		path = path + "?" + r.values.Encode()
	}
	u, err := url.Parse(s.URL + path)
	if err != nil {
		r.t.Error(err)
		r.t.FailNow()
	}
	var req *http.Request
	req, err = http.NewRequest(r.method, u.String(), r.body)
	req.Header = r.header
	if err != nil {
		r.t.Error(err)
		r.t.FailNow()
	}
	client := &http.Client{}
	var res *http.Response
	res, err = client.Do(req)
	if err != nil {
		r.t.Error(err)
		r.t.FailNow()
	}
	return res, err
}

// reset resets the state of the HttpTestRunner struct, clearing body, handler
// function, header, and values if their corresponding clear flags are set to
// true.
func (r *HttpTestRunner) reset() {
	if r.clearBody {
		r.body = nil
		r.clearBody = false
	}
	if r.clearHandlerFunc {
		r.handlerFunc = nil
		r.clearHandlerFunc = false
	}
	if r.clearHeader {
		r.header = http.Header{}
		r.clearHeader = false
	}
	if r.clearValues {
		r.values = url.Values{}
		r.clearValues = false
	}
}

// Run executes the HTTP request method specified by the HttpTestRunner struct
// and returns the response and an error if any occurred during the execution.
func (r *HttpTestRunner) Run() (res *http.Response, err error) {
	defer r.reset()
	switch r.method {
	case http.MethodDelete, http.MethodGet, http.MethodHead, http.MethodPatch,
		http.MethodPost, http.MethodPut:
		res, err = r.runMethod()
	default:
		res, err = nil, errors.New(
			fmt.Sprintf("unsupported method: %s", r.method))
	}
	return res, err
}

// resetMethod resets the HTTP method of the HttpTestRunner struct to the
// specified previous method if it is different from the current method.
func (r *HttpTestRunner) resetMethod(previous string) {
	if previous != r.method {
		r.method = previous
	}
}

// Delete executes an HTTP DELETE request using HttpTestRunner.Run and returns
// the response and an error if any occurred during the execution.
//
// It will reset to the previous method in case if it wasn't http.MethodDelete.
func (r *HttpTestRunner) Delete() (res *http.Response, err error) {
	previousMethod := r.method
	r.method = http.MethodDelete
	defer r.resetMethod(previousMethod)
	r.method = http.MethodDelete
	res, err = r.Run()
	return res, err
}

// Get executes an HTTP GET request using HttpTestRunner.Run and returns
// the response and an error if any occurred during the execution.
//
// It will reset to the previous method in case if it wasn't http.MethodGet.
func (r *HttpTestRunner) Get() (res *http.Response, err error) {
	previousMethod := r.method
	r.method = http.MethodGet
	defer r.resetMethod(previousMethod)
	res, err = r.Run()
	return res, err
}

// Head executes an HTTP HEAD request using HttpTestRunner.Run and returns
// the response and an error if any occurred during the execution.
//
// It will reset to the previous method in case if it wasn't http.MethodHead.
func (r *HttpTestRunner) Head() (res *http.Response, err error) {
	previousMethod := r.method
	r.method = http.MethodHead
	defer r.resetMethod(previousMethod)
	res, err = r.Run()
	return res, err
}

// Patch executes an HTTP PATCH request using HttpTestRunner.Run and returns
// the response and an error if any occurred during the execution.
//
// It will reset to the previous method in case if it wasn't http.MethodPatch.
func (r *HttpTestRunner) Patch() (res *http.Response, err error) {
	previousMethod := r.method
	r.method = http.MethodPatch
	defer r.resetMethod(previousMethod)
	res, err = r.Run()
	return res, err
}

// Post executes an HTTP POST request using HttpTestRunner.Run and returns
// the response and an error if any occurred during the execution.
//
// It will reset to the previous method in case if it wasn't http.MethodPost.
func (r *HttpTestRunner) Post() (res *http.Response, err error) {
	previousMethod := r.method
	r.method = http.MethodPost
	defer r.resetMethod(previousMethod)
	res, err = r.Run()
	return res, err
}

// Put executes an HTTP PUT request using HttpTestRunner.Run and returns
// the response and an error if any occurred during the execution.
//
// It will reset to the previous method in case if it wasn't http.MethodPut.
func (r *HttpTestRunner) Put() (res *http.Response, err error) {
	previousMethod := r.method
	r.method = http.MethodPut
	defer r.resetMethod(previousMethod)
	res, err = r.Run()
	return res, err
}

// BodyAsString returns the body of a request as string
func BodyAsString(t *testing.T, res *http.Response) string {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	return string(body)
}

// BodyAsJson unmarshal the body of a request to json
func BodyAsJson(t *testing.T, res *http.Response, jsonBody any) {
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	err = json.Unmarshal(b, jsonBody)
	if err != nil {
		t.Error(err)
	}
}

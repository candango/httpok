package testrunner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
)

type TargetHandler struct {
	http.Handler
	log.Logger
}

func NewTargetHandler() *TargetHandler {
	h := http.NewServeMux()
	t := &TargetHandler{
		Handler: h,
	}
	h.Handle("/get", &GetHandler{})
	h.Handle("/head", &HeadHandler{})
	h.Handle("/post", &PostHandler{})
	h.Handle("/postJson", &PostJsonHandler{})
	return t
}

type TargetRequest struct {
	Data TargetData `json:"data"`
}

type TargetData struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type TargetResponse struct {
	Code    int
	Message string `json:"message"`
}

func handlerFunc(res http.ResponseWriter, req *http.Request) {
	targetResponse := &TargetResponse{Message: "OK from handler func"}
	data, err := json.Marshal(targetResponse)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = res.Write(data)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type GetHandler struct{}

func (h *GetHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	method := req.Method
	if method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	targetResponse := &TargetResponse{Message: "OK"}
	selector := req.URL.Query().Get("selector")
	if selector == "bad" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	data, err := json.Marshal(targetResponse)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = res.Write(data)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type HeadHandler struct{}

func (h *HeadHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	method := req.Method
	if method != http.MethodHead {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	selector := req.Header.Get("selector")
	if selector == "bad" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	res.Header().Add("return", "OK")
}

type PostHandler struct{}

func (h *PostHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	method := req.Method
	if method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	targetResponse := &TargetResponse{Message: "OK"}
	body, err := readBody(req)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	if body == "bad" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	data, err := json.Marshal(targetResponse)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = res.Write(data)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type PostJsonHandler struct{}

func (h *PostJsonHandler) ServeHTTP(res http.ResponseWriter,
	req *http.Request) {
	method := req.Method
	if method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	targetResponse := &TargetResponse{Message: "OK"}

	body := &TargetRequest{}
	status := processJsonRequest(req, body)
	if status != 200 {
		res.WriteHeader(status)
		return
	}
	targetResponse.Message = fmt.Sprintf(
		"id: %s, type: %s", body.Data.Id, body.Data.Type)
	data, err := json.Marshal(targetResponse)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = res.Write(data)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func readBody(request *http.Request) (string, error) {
	body, err := io.ReadAll(request.Body)
	return string(body), err
}

func processJsonRequest(req *http.Request, a any) (statusCode int) {
	if strings.ToLower(req.Header.Get("Content-Type")) != "application/json" {
		return http.StatusBadRequest
	}
	b, _ := io.ReadAll(req.Body)
	err := json.Unmarshal(b, a)
	if err != nil {
		return http.StatusInternalServerError
	}
	return 200
}

func TestWithHandler(t *testing.T) {
	runner := NewHttpTestRunner(t).WithHandler(NewTargetHandler())

	t.Run("With func, clear func after", func(t *testing.T) {
		res, err := runner.WithHandlerFunc(
			handlerFunc).ClearHandlerFuncAfter().Get()
		if err != nil {
			t.Error(err)
		}
		jsonBody := &TargetResponse{}
		BodyAsJson(t, res, jsonBody)
		assert.Equal(t, "200 OK", res.Status)
		assert.Equal(t, jsonBody.Message, "OK from handler func")
	})

	t.Run("Get Request tests", func(t *testing.T) {
		t.Run("Method not allowed", func(t *testing.T) {
			res, err := runner.WithPath("/post").Get()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, "405 Method Not Allowed", res.Status)
		})

		t.Run("Request OK", func(t *testing.T) {
			values := url.Values{}
			values.Add("selector", "ok")
			res, err := runner.WithPath("/get").WithValues(values).Get()
			if err != nil {
				t.Error(err)
			}
			jsonBody := &TargetResponse{}
			BodyAsJson(t, res, jsonBody)
			assert.Equal(t, "200 OK", res.Status)
			assert.Equal(t, jsonBody.Message, "OK")
		})

		t.Run("Bad request", func(t *testing.T) {
			values := url.Values{}
			values.Add("selector", "bad")
			res, err := runner.WithPath("/get").WithValues(values).Get()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, "400 Bad Request", res.Status)
		})

		t.Run("Method not allowed", func(t *testing.T) {
			res, err := runner.WithPath("/post").Get()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, "405 Method Not Allowed", res.Status)
		})
	})

	t.Run("Head Request tests", func(t *testing.T) {
		t.Run("Method not allowed", func(t *testing.T) {
			res, err := runner.WithPath("/head").Post()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, "405 Method Not Allowed", res.Status)
			assert.Equal(t, runner.method, http.MethodGet)
		})

		t.Run("Request OK", func(t *testing.T) {
			res, err := runner.WithPath("/head").Head()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, "200 OK", res.Status)
			assert.Equal(t, http.NoBody, res.Body)
			assert.Equal(t, "OK", res.Header.Get("return"))
		})

		t.Run("Bad request, clear header after", func(t *testing.T) {
			res, err := runner.ClearHeaderAfter().WithHeader(
				"selector", "bad").WithPath("/head").Head()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, "400 Bad Request", res.Status)
			assert.Equal(t, http.NoBody, res.Body)
			assert.Equal(t, http.Header{}, runner.header)
		})
	})

	// not testing put as it acts as post
	t.Run("Post Request tests", func(t *testing.T) {
		t.Run("Method not allowed", func(t *testing.T) {
			res, err := runner.WithPath("/get").Head()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, "405 Method Not Allowed", res.Status)
			assert.Equal(t, runner.method, http.MethodGet)
			assert.Equal(t, http.NoBody, res.Body)
		})

		t.Run("Request OK", func(t *testing.T) {
			res, err := runner.WithPath("/post").Post()
			if err != nil {
				t.Error(err)
			}
			jsonBody := &TargetResponse{}
			BodyAsJson(t, res, jsonBody)
			assert.Equal(t, "200 OK", res.Status)
			assert.Equal(t, "OK", jsonBody.Message)

		})

		t.Run("Bad request, clear body after ", func(t *testing.T) {
			res, err := runner.ClearBodyAfter().WithStringBody(
				"bad").WithPath("/post").Post()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, "400 Bad Request", res.Status)
			assert.Equal(t, nil, runner.body)
		})

		t.Run("With json body test", func(t *testing.T) {
			jsonRequest := &TargetRequest{TargetData{
				Id:   "1",
				Type: "a",
			}}
			runner.ClearBodyAfter().ClearHeaderAfter()
			runner.WithHeader("Content-Type", "application/json")
			runner.WithJsonBody(jsonRequest)
			res, err := runner.WithPath("/postJson").Post()
			if err != nil {
				t.Error(err)
			}
			jsonBody := &TargetResponse{}
			BodyAsJson(t, res, jsonBody)
			assert.Equal(t, "200 OK", res.Status)
			assert.Equal(t, "id: 1, type: a", jsonBody.Message)
			assert.Equal(t, nil, runner.body)
			assert.Equal(t, http.Header{}, runner.header)
		})

		t.Run("With json body missing json header", func(t *testing.T) {
			jsonRequest := &TargetRequest{TargetData{
				Id:   "1",
				Type: "a",
			}}
			runner.ClearBodyAfter().WithJsonBody(jsonRequest)
			res, err := runner.WithPath("/postJson").Post()
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, "400 Bad Request", res.Status)
		})
	})
}

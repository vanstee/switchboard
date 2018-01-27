package switchboard_test

import (
	"io"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/vanstee/switchboard"
)

func TestExecuteSimpleCommand(t *testing.T) {
	route := &switchboard.Route{
		Path: "/users",
		Command: &switchboard.Command{
			Driver: &FakeDriver{
				Stdout: `HTTP_CONTENT_TYPE: application/json
HTTP_STATUS_CODE: 201

{ "user": { "id": 1, "name": "Jimmy" } }`,
			},
		},
		Methods: []string{"POST"},
	}

	req := httptest.NewRequest(
		"POST",
		"http://example.com/users",
		strings.NewReader(`{ "user": { "name": "Jimmy" } }`),
	)
	w := httptest.NewRecorder()

	pipeline := switchboard.Pipeline{route}
	pipeline.Handle(w, req, make(httprouter.Params, 0))

	resp := w.Result()
	if resp.StatusCode != 201 {
		t.Errorf("excepted response status to be %d, got %d", 201, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("excepted Content-Type header to equal %s, got %s", "application/json", contentType)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll returned an error: %s", err)
	}
	if string(body) != "{ \"user\": { \"id\": 1, \"name\": \"Jimmy\" } }\n" {
		t.Errorf("expected response body was incorrect")
	}
}

type FakeDriver struct {
	Stdout string
	Stderr string
	Status int64
	Err    error
}

func (driver FakeDriver) Execute(command *switchboard.Command, env []string, streams *switchboard.Streams) (int64, error) {
	var err error
	_, err = io.WriteString(streams.Stdout, driver.Stdout)
	if err != nil {
		return -1, err
	}
	_, err = io.WriteString(streams.Stderr, driver.Stderr)
	if err != nil {
		return -1, err
	}

	return driver.Status, driver.Err
}

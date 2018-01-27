package switchboard_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/vanstee/switchboard"
)

var (
	port = 8081

	exampleTests = []struct {
		path     string
		reqresps []struct {
			req  *http.Request
			resp *http.Response
		}
	}{
		{
			path: "examples/authentication.yaml",
			reqresps: []struct {
				req  *http.Request
				resp *http.Response
			}{
				{
					req: &http.Request{
						Method: "GET",
						URL:    URLMustParse("http://localhost:8081/user"),
					},
					resp: &http.Response{
						StatusCode: 401,
						Header: map[string][]string{
							"Content-Type": []string{
								"application/json",
							},
						},
						Body: ioutil.NopCloser(
							strings.NewReader(
								"{ \"error\": \"Unauthorized\" }\n",
							),
						),
					},
				},
				{
					req: &http.Request{
						Method: "GET",
						URL:    URLMustParse("http://localhost:8081/user"),
						Header: map[string][]string{
							"Authorization": []string{
								"secret",
							},
						},
					},
					resp: &http.Response{
						StatusCode: 200,
						Header: map[string][]string{
							"Content-Type": []string{
								"application/json",
							},
						},
						Body: ioutil.NopCloser(
							strings.NewReader(
								"{ \"user\": { \"id\": 1 } }\n",
							),
						),
					},
				},
			},
		},
		{
			path: "examples/nesting.yaml",
			reqresps: []struct {
				req  *http.Request
				resp *http.Response
			}{
				{
					req: &http.Request{
						Method: "GET",
						URL:    URLMustParse("http://localhost:8081/articles/1/comments/1"),
					},
					resp: &http.Response{
						StatusCode: 200,
					},
				},
				{
					req: &http.Request{
						Method: "GET",
						URL:    URLMustParse("http://localhost:8081/articles/1"),
					},
					resp: &http.Response{
						StatusCode: 404,
					},
				},
				{
					req: &http.Request{
						Method: "GET",
						URL:    URLMustParse("http://localhost:8081/missing"),
					},
					resp: &http.Response{
						StatusCode: 404,
					},
				},
			},
		},
		{
			path: "examples/restful.yaml",
			reqresps: []struct {
				req  *http.Request
				resp *http.Response
			}{
				{
					req: &http.Request{
						Method: "GET",
						URL:    URLMustParse("http://localhost:8081/users"),
					},
					resp: &http.Response{
						StatusCode: 200,
						Header: map[string][]string{
							"Content-Type": []string{
								"application/json",
							},
						},
						Body: ioutil.NopCloser(
							strings.NewReader(
								"{\"users\":[]}\n",
							),
						),
					},
				},
			},
		},
	}
)

func TestExamples(t *testing.T) {
	for _, test := range exampleTests {
		server, err := switchboard.NewServer(test.path, port, false)
		if err != nil {
			t.Fatalf("NewServer returned an error: %s", err)
		}

		sdone := make(chan struct{})
		serrc := make(chan error)
		go func() {
			err = server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				t.Logf("ListenAndServe returned an error: %s", err)
				serrc <- err
				return
			}

			close(sdone)
		}()

		cdone := make(chan struct{})
		cerrc := make(chan error)
		go func() {
			for _, reqresp := range test.reqresps {
				resp, err := http.DefaultClient.Do(reqresp.req)
				if err != nil {
					t.Logf("%s %s returned an error: %s", reqresp.req.Method, reqresp.req.URL, err)
					cerrc <- err
					return
				}

				if resp.StatusCode != reqresp.resp.StatusCode {
					t.Errorf(
						"%s %s excepted response status to be %d, got %d",
						reqresp.req.Method,
						reqresp.req.URL,
						reqresp.resp.StatusCode,
						resp.StatusCode,
					)
				}

				for key, tvalue := range reqresp.resp.Header {
					value, _ := resp.Header[key]
					if value[0] != tvalue[0] {
						t.Errorf(
							"%s %s excepted %s header to equal %s, got %s",
							reqresp.req.Method,
							reqresp.req.URL,
							key,
							tvalue[0],
							value[0],
						)
					}
				}

				if reqresp.resp.Body == nil {
					continue
				}

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Logf("ioutil.ReadAll returned an error: %s", err)
					cerrc <- err
					return
				}

				tbody, err := ioutil.ReadAll(reqresp.resp.Body)
				if err != nil {
					t.Logf("ioutil.ReadAll returned an error: %s", err)
					cerrc <- err
					return
				}

				if string(body) != string(tbody) {
					t.Errorf(
						"%s %s expected response body was incorrect",
						reqresp.req.Method,
						reqresp.req.URL,
					)
				}
			}

			close(cdone)
		}()

		select {
		case <-cerrc:
			t.Fail()
		case <-cdone:
			// do nothing
		}

		server.Shutdown(nil)

		select {
		case <-serrc:
			t.Fail()
		case <-sdone:
			// do nothing
		}
	}
}

func URLMustParse(str string) *url.URL {
	url, err := url.Parse(str)
	if err != nil {
		panic(fmt.Sprintf("url: Parse(%v): %s", str, err))
	}
	return url
}

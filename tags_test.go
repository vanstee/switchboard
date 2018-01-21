package switchboard_test

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/vanstee/switchboard"
)

var (
	tagTests = []struct {
		stdout string
		tags   switchboard.Tags
		rest   string
	}{
		{
			stdout: `
HTTP_CONTENT_TYPE: application/json
HTTP_STATUS_CODE: 201

{ "user": { "name": "Patrick" } }`,
			tags: switchboard.Tags{
				"HTTP_CONTENT_TYPE": []string{"application/json"},
				"HTTP_STATUS_CODE":  []string{"201"},
			},
			rest: "{ \"user\": { \"name\": \"Patrick\" } }\n",
		},
	}
)

func TestParseTags(t *testing.T) {
	for _, test := range tagTests {
		tags, restr, err := switchboard.ParseTags(strings.NewReader(test.stdout))
		if err != nil {
			t.Fatalf("ParseTags returned an error: %s", err)
		}

		if len(tags) != len(test.tags) {
			t.Fatalf("expected %d tags, got %d tags", len(test.tags), len(tags))
		}
		for name, tvalues := range test.tags {
			values, ok := tags[name]
			if !ok {
				t.Errorf("tag %s not found", name)
			}

			for i, tvalue := range tvalues {
				if values[i] != tvalue {
					t.Errorf("expected tag %s at index %d to be %s, got %s", name, i, tvalue, values[i])
				}
			}
		}

		rest, err := ioutil.ReadAll(restr)
		if err != nil {
			t.Fatalf("reading from reader returned an error: %s", err)
		}
		if string(rest) != test.rest {
			t.Errorf("expected rest to be %#v, got %#v", test.rest, string(rest))
		}
	}
}

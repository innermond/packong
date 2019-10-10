package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_fitboxes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(fitboxes))
	defer ts.Close()

	ts.URL += API_PATH

	type test struct {
		data   string
		status int
	}

	t.Run("invalid input", func(t *testing.T) {

		tt := []test{
			{`{"width":500,"height":500,"dimensions":["501x"]}`, 422},
			{`{"width":500,"height":500,"dimensions":[]}`, 422},
			{`{"width":500,"height":500,"dimensions":["qwqw"]}`, 422},
			{`{"width":500,"height":500,"dimensions":"500x500"}`, 422},
			{`{"width":500,"height":500}`, 422},
			{`{"width":500,"height":500,"dimensions":["501x"]}`, 422},
			{`{"width":"50x","height":"x00"}`, 422},
			{`{}`, 422},
		}
		var buf *bytes.Buffer

		for _, tc := range tt {
			buf = bytes.NewBuffer([]byte(tc.data))

			resp := post(t, ts.URL+"/", buf)
			defer resp.Body.Close()

			if resp.StatusCode != tc.status {
				t.Errorf("got status %d, expected %d", resp.StatusCode, tc.status)
			}
			//t.Log(tc.data, http.StatusText(tc.status))
		}
	})

	t.Run("malformed json", func(t *testing.T) {

		tt := []test{
			{`{"width":aaa,"height":500}`, 400},
			{`{"width":,"height":500}`, 400},
		}
		var buf *bytes.Buffer

		for _, tc := range tt {
			buf = bytes.NewBuffer([]byte(tc.data))

			resp := post(t, ts.URL+"/", buf)
			defer resp.Body.Close()

			if resp.StatusCode != tc.status {
				t.Errorf("got status %d, expected %d", resp.StatusCode, tc.status)
			}
			//t.Log(tc.data, http.StatusText(tc.status))
		}
	})

	t.Run("valid input", func(t *testing.T) {

		tt := []test{
			{`{"width":1540,"height":50000,"dimensions":["500x1500x10"]}`, 200},
			{`{"width":1270,"height":50000,"dimensions":["500x1200x10","780x650x3","890x1300"]}`, 200},
			{`{"width":500,"height":500,"dimensions":["501x501"]}`, 200},
		}
		var buf *bytes.Buffer

		for _, tc := range tt {
			buf = bytes.NewBuffer([]byte(tc.data))

			resp := post(t, ts.URL+"/", buf)
			defer resp.Body.Close()

			if resp.StatusCode != tc.status {
				t.Errorf("got status %d, expected %d", resp.StatusCode, tc.status)
			}

			buf.Reset()
			_, err := io.CopyN(buf, resp.Body, 150)
			if err != nil {
				t.Fatal(err)
			}

			if buf.Len() == 0 {
				t.Errorf("data %s has unexpected zero body response", tc.data)
			}

			t.Log(tc.data, buf.String())
		}
	})

}

func post(t *testing.T, url string, buf *bytes.Buffer) *http.Response {
	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

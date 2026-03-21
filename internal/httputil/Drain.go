package httputil

import (
	"io"
	"io/ioutil"
)

// Drain and close http.Body
// this function to help reuse http.Client connections by discard and close unwanted response Body
func DrainAndClose(reader io.ReadCloser) (err error) {
	_, err = io.Copy(ioutil.Discard, reader)
	if err != nil {
		return
	}
	return reader.Close()
}

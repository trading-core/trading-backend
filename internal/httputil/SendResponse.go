package httputil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/internal/logger"
)

func SendErrorResponse(responseWriter http.ResponseWriter, err error) {
	// get the message to give the user:
	message := merry.UserMessage(err)
	if message == "" {
		message = merry.Message(err)
	}
	statusCode := merry.HTTPCode(err)
	// Don't just call SendResponseMessage:
	// we want the warning to print the location in our parent frame.
	logger.Warnpf("%d, %s", statusCode, err.Error())
	SendResponseJSON(responseWriter, statusCode, Message{Message: message})
}

func SendResponseJSON(responseWriter http.ResponseWriter, statusCode int, body interface{}) {
	bytes, err := json.Marshal(body)
	if err == nil {
		responseWriter.Header().Set("Content-Type", "application/json; charset=UTF-8")
		responseWriter.WriteHeader(statusCode)
		responseWriter.Write(bytes)
	} else {
		panic("failed to marshal json")
	}
}

func ExtractResponseError(response *http.Response) error {
	data, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	return fmt.Errorf("failed to perform request: %s", string(data))
}

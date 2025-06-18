package fatal

import (
	"encoding/json"
	"io"
	"tradingbot/internal/logger"
)

// A set of methods that encapsulate actions that will simply fatal on error.
// Typically we call them Unless<action>, because the call
//     fatal.Unless<Action>(args)
// tells the story

// Unless fatal on condition failure
func Unless(b bool, args ...interface{}) {
	if !b {
		logger.Fatal(args...)
	}
}

// Unlessf fatal on condition failure
func Unlessf(b bool, format string, args ...interface{}) {
	if !b {
		logger.Fatalf(format, args...)
	}
}

// UnlessMarshal marshal value, fatal on error
func UnlessMarshal(v interface{}) (b []byte) {
	b, err := json.Marshal(v)
	OnError(err, "marshalling")
	return b
}

// UnlessUnmarshal unmarshal value, fatal on error
func UnlessUnmarshal(data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	OnError(err, "unmarshalling")
}

// UnlessDecode decode value, fatal on error
func UnlessDecode(reader io.Reader, v interface{}) {
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	err := decoder.Decode(v)
	OnError(err, "decoding")
}

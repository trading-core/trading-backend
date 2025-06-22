package fatal

import (
	"fmt"

	"github.com/kduong/tradingbot/internal/logger"
)

// OnError fatal on non-nil error
func OnError(err error, args ...interface{}) {
	if err == nil {
		return
	}
	if len(args) == 0 {
		logger.Fatal(err.Error())
	}
	context := fmt.Sprint(args...)
	logger.Fatalf("%s: %s", context, err.Error())
}

// OnErrorf fatal on non-nil error
func OnErrorf(err error, context string, args ...interface{}) {
	if err == nil {
		return
	}
	if len(args) > 0 {
		context = fmt.Sprintf(context, args...)
	}
	logger.Fatalf("%s: %s", context, err.Error())
}

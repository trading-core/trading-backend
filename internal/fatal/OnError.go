package fatal

import (
	"context"
	"fmt"

	"github.com/kduong/trading-backend/internal/logger"
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

// OnErrorUnlessDone panics or logs fatal if err is non-nil
// unless the context has already been canceled.
func OnErrorUnlessDone(ctx context.Context, err error) {
	if err == nil {
		return
	}
	select {
	case <-ctx.Done():
		// Context canceled, ignore the error
		return
	default:
		// Context active, treat error as fatal
		logger.Fatal(err)
	}
}

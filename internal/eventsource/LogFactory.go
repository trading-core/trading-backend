package eventsource

import (
	"fmt"
	"io"
	"strings"

	"github.com/kduong/trading-backend/internal/config"
)

type LogFactory interface {
	// Close releases resources owned by the factory.
	// For shared backends (for example Redis), the factory owns the client lifecycle
	// and should close it. Logs created by the factory should not close shared clients.
	// In-memory factories may implement this as a no-op.
	io.Closer

	// Create a log for the given channel.
	Create(channel string) (log Log, err error)
}

func LogFactoryFromEnv(prefix string, dflt string) (factory LogFactory, err error) {
	implementation := strings.ToUpper(config.EnvString(prefix+"_FACTORY", dflt))
	keyPrefix := prefix + "_" + implementation
	switch implementation {
	case "DB":
		panic("DB log factory not implemented")
	case "REDIS":
		factory = NewRedisLogFactory(NewRedisLogFactoryInput{
			Address: config.EnvStringOrFatal(keyPrefix + "_ADDRESS"),
		})
		return
	case "INMEMORY":
		factory = NewInMemoryLogFactory()
		return
	default:
		err = fmt.Errorf("LogFactory %s not supported", implementation)
		return
	}
}

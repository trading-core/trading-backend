package broker

import (
	"context"
)

type AccountDiscoveryClient interface {
	ListAccountIDs(ctx context.Context) ([]string, error)
}

package port

import "context"

type Port interface {
	Start() error
	Stop(ctx context.Context) error
}

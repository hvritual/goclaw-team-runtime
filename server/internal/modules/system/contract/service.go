package contract

import "context"

type Service interface {
	Ping(context.Context) (string, error)
}

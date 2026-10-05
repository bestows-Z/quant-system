package lock

import "context"

type Locker interface {
	Lock(ctx context.Context, key string) (unlock func(), err error)
}

package routine

import "context"

func Recover(cleanups ...func(r interface{})) {
	if r := recover(); r != nil {
		for _, cleanup := range cleanups {
			cleanup(r)
		}
	}
}

func RecoverCtx(ctx context.Context, cleanups ...func(ctx context.Context, r interface{})) {
	if r := recover(); r != nil {
		for _, cleanup := range cleanups {
			cleanup(ctx, r)
		}
	}
}

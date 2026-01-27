package auth

import "context"

type ctxKey int

const authCtxKey ctxKey = iota

func WithAuthContext(ctx context.Context, a AuthContext) context.Context {
    return context.WithValue(ctx, authCtxKey, a)
}

func FromContext(ctx context.Context) (AuthContext, bool) {
    v := ctx.Value(authCtxKey)
    if v == nil {
        return AuthContext{}, false
    }
    a, ok := v.(AuthContext)
    return a, ok
}

package access

import "context"

const BootstrapOwnerID = "00000000-0000-4000-8000-000000000001"

type Principal struct {
	ID   string
	Name string
	Role string
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func Current(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok && p.ID != ""
}

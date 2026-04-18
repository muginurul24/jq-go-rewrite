package auth

import "context"

type contextKey string

const (
	currentUserKey contextKey = "currentUser"
	currentTokoKey contextKey = "currentToko"
)

func WithCurrentUser(ctx context.Context, user PublicUser) context.Context {
	return context.WithValue(ctx, currentUserKey, user)
}

func CurrentUser(ctx context.Context) (PublicUser, bool) {
	user, ok := ctx.Value(currentUserKey).(PublicUser)
	return user, ok
}

func WithCurrentToko(ctx context.Context, toko Toko) context.Context {
	return context.WithValue(ctx, currentTokoKey, toko)
}

func CurrentToko(ctx context.Context) (Toko, bool) {
	toko, ok := ctx.Value(currentTokoKey).(Toko)
	return toko, ok
}

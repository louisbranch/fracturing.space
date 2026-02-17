package auth

import (
	"context"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userCreator struct {
	store       storage.UserStore
	clock       func() time.Time
	idGenerator func() (string, error)
}

// userCreator isolates user bootstrap policy (id/time strategy and persistence) from
// the transport handler, so gRPC methods only coordinate request input and response.
func newUserCreator(service *AuthService) userCreator {
	creator := userCreator{store: service.store, clock: service.clock, idGenerator: service.idGenerator}
	if creator.clock == nil {
		creator.clock = time.Now
	}
	return creator
}

// create builds a new user aggregate through domain primitives and stores it in the
// configured user store, ensuring user creation remains a domain decision plus
// persistence side effect rather than duplicated gRPC command logic.
func (c userCreator) create(ctx context.Context, in *authv1.CreateUserRequest) (user.User, error) {
	if c.store == nil {
		return user.User{}, status.Error(codes.Internal, "user store is not configured")
	}

	created, err := user.CreateUser(user.CreateUserInput{
		Username: in.GetUsername(),
		Locale:   in.GetLocale(),
	}, c.clock, c.idGenerator)
	if err != nil {
		return user.User{}, err
	}

	if err := c.store.PutUser(ctx, created); err != nil {
		return user.User{}, status.Errorf(codes.Internal, "put user: %v", err)
	}

	return created, nil
}

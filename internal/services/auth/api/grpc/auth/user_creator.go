package auth

import (
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userCreator struct {
	storeConfigured bool
	clock           func() time.Time
	idGenerator     func() (string, error)
}

// userCreator isolates user bootstrap policy (id/time strategy and persistence) from
// the transport handler, so gRPC methods only coordinate request input and response.
func newUserCreator(service *AuthService) userCreator {
	creator := userCreator{storeConfigured: service.store != nil, clock: service.clock, idGenerator: service.idGenerator}
	if creator.clock == nil {
		creator.clock = time.Now
	}
	return creator
}

// create builds a new user aggregate through domain primitives.
//
// Persistence is intentionally deferred so callers can compose the user write
// with transactional outbox persistence.
func (c userCreator) create(in *authv1.CreateUserRequest) (user.User, error) {
	if !c.storeConfigured {
		return user.User{}, status.Error(codes.Internal, "user store is not configured")
	}

	created, err := user.CreateUser(user.CreateUserInput{
		Email: in.GetEmail(),
	}, c.clock, c.idGenerator)
	if err != nil {
		return user.User{}, err
	}
	return created, nil
}

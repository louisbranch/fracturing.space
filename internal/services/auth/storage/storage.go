package storage

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = errors.New(errors.CodeNotFound, "record not found")

// UserStore persists auth user records.
type UserStore interface {
	PutUser(ctx context.Context, u user.User) error
	GetUser(ctx context.Context, userID string) (user.User, error)
	ListUsers(ctx context.Context, pageSize int, pageToken string) (UserPage, error)
}

// UserPage describes a page of user records.
type UserPage struct {
	Users         []user.User
	NextPageToken string
}

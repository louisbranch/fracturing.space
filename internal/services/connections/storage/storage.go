// Package storage defines persistence contracts for connections service state.
package storage

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound indicates a requested contact record is missing.
var ErrNotFound = errors.New("record not found")

// ErrAlreadyExists indicates a uniqueness-constrained record already exists.
var ErrAlreadyExists = errors.New("record already exists")

// Contact stores one owner-scoped directed contact relationship.
type Contact struct {
	OwnerUserID   string
	ContactUserID string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ContactPage stores a page of directed contacts.
type ContactPage struct {
	Contacts      []Contact
	NextPageToken string
}

// UserProfileRecord stores one social/discovery profile for a user.
type UserProfileRecord struct {
	UserID        string
	Username      string
	Name          string
	AvatarSetID   string
	AvatarAssetID string
	Bio           string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ContactStore persists owner-scoped directed contact relationships.
type ContactStore interface {
	PutContact(ctx context.Context, contact Contact) error
	GetContact(ctx context.Context, ownerUserID string, contactUserID string) (Contact, error)
	DeleteContact(ctx context.Context, ownerUserID string, contactUserID string) error
	ListContacts(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (ContactPage, error)
}

// UserProfileStore persists social/discovery user profile records.
type UserProfileStore interface {
	PutUserProfile(ctx context.Context, profile UserProfileRecord) error
	GetUserProfileByUserID(ctx context.Context, userID string) (UserProfileRecord, error)
	GetUserProfileByUsername(ctx context.Context, username string) (UserProfileRecord, error)
}

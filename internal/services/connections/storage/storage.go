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

// UsernameRecord stores one canonical username claim for a user.
type UsernameRecord struct {
	UserID    string
	Username  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PublicProfileRecord stores one public profile for user identity verification.
type PublicProfileRecord struct {
	UserID        string
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

// UsernameStore persists canonical username claims.
type UsernameStore interface {
	PutUsername(ctx context.Context, username UsernameRecord) error
	GetUsernameByUserID(ctx context.Context, userID string) (UsernameRecord, error)
	GetUsernameByUsername(ctx context.Context, username string) (UsernameRecord, error)
}

// ProfileStore persists public profile records.
type ProfileStore interface {
	PutPublicProfile(ctx context.Context, profile PublicProfileRecord) error
	GetPublicProfileByUserID(ctx context.Context, userID string) (PublicProfileRecord, error)
}

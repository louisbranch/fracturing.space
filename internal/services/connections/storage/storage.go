// Package storage defines persistence contracts for connections service state.
package storage

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound indicates a requested contact record is missing.
var ErrNotFound = errors.New("record not found")

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

// ContactStore persists owner-scoped directed contact relationships.
type ContactStore interface {
	PutContact(ctx context.Context, contact Contact) error
	GetContact(ctx context.Context, ownerUserID string, contactUserID string) (Contact, error)
	DeleteContact(ctx context.Context, ownerUserID string, contactUserID string) error
	ListContacts(ctx context.Context, ownerUserID string, pageSize int, pageToken string) (ContactPage, error)
}

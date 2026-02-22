package connections

import (
	"context"
	"errors"
	"strings"
	"time"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/connections/storage"
	usernameutil "github.com/louisbranch/fracturing.space/internal/services/connections/username"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListContactsPageSize = 10
	maxListContactsPageSize     = 50
)

type contactAndUsernameStore interface {
	storage.ContactStore
	storage.UsernameStore
}

// Service exposes connections.v1 gRPC operations.
type Service struct {
	connectionsv1.UnimplementedConnectionsServiceServer
	store contactAndUsernameStore
	clock func() time.Time
}

// NewService creates a connections service backed by contact storage.
func NewService(store contactAndUsernameStore) *Service {
	return &Service{
		store: store,
		clock: time.Now,
	}
}

// AddContact adds one owner-scoped directed contact relationship.
func (s *Service) AddContact(ctx context.Context, in *connectionsv1.AddContactRequest) (*connectionsv1.AddContactResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "add contact request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}

	ownerUserID := strings.TrimSpace(in.GetOwnerUserId())
	contactUserID := strings.TrimSpace(in.GetContactUserId())
	if ownerUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "owner user id is required")
	}
	if contactUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "contact user id is required")
	}
	if ownerUserID == contactUserID {
		return nil, status.Error(codes.InvalidArgument, "contact user id must differ from owner user id")
	}

	now := time.Now()
	if s.clock != nil {
		now = s.clock()
	}
	contact := storage.Contact{
		OwnerUserID:   ownerUserID,
		ContactUserID: contactUserID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.store.PutContact(ctx, contact); err != nil {
		return nil, status.Errorf(codes.Internal, "add contact: %v", err)
	}
	persisted, err := s.store.GetContact(ctx, ownerUserID, contactUserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add contact: %v", err)
	}
	return &connectionsv1.AddContactResponse{
		Contact: contactToProto(persisted),
	}, nil
}

// RemoveContact removes one owner-scoped directed contact relationship.
func (s *Service) RemoveContact(ctx context.Context, in *connectionsv1.RemoveContactRequest) (*connectionsv1.RemoveContactResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "remove contact request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}
	ownerUserID := strings.TrimSpace(in.GetOwnerUserId())
	contactUserID := strings.TrimSpace(in.GetContactUserId())
	if ownerUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "owner user id is required")
	}
	if contactUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "contact user id is required")
	}
	if ownerUserID == contactUserID {
		return nil, status.Error(codes.InvalidArgument, "contact user id must differ from owner user id")
	}

	if err := s.store.DeleteContact(ctx, ownerUserID, contactUserID); err != nil {
		return nil, status.Errorf(codes.Internal, "remove contact: %v", err)
	}
	return &connectionsv1.RemoveContactResponse{}, nil
}

// ListContacts returns one page of owner-scoped directed contacts.
func (s *Service) ListContacts(ctx context.Context, in *connectionsv1.ListContactsRequest) (*connectionsv1.ListContactsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list contacts request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}

	ownerUserID := strings.TrimSpace(in.GetOwnerUserId())
	if ownerUserID == "" {
		return nil, status.Error(codes.InvalidArgument, "owner user id is required")
	}
	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListContactsPageSize,
		Max:     maxListContactsPageSize,
	})
	page, err := s.store.ListContacts(ctx, ownerUserID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list contacts: %v", err)
	}

	resp := &connectionsv1.ListContactsResponse{
		Contacts:      make([]*connectionsv1.Contact, 0, len(page.Contacts)),
		NextPageToken: page.NextPageToken,
	}
	for _, contact := range page.Contacts {
		resp.Contacts = append(resp.Contacts, contactToProto(contact))
	}
	return resp, nil
}

// SetUsername claims or updates one canonical username for a user.
func (s *Service) SetUsername(ctx context.Context, in *connectionsv1.SetUsernameRequest) (*connectionsv1.SetUsernameResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "set username request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}
	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	canonicalUsername, err := usernameutil.Canonicalize(in.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "username is invalid: %v", err)
	}

	now := time.Now()
	if s.clock != nil {
		now = s.clock()
	}
	if err := s.store.PutUsername(ctx, storage.UsernameRecord{
		UserID:    userID,
		Username:  canonicalUsername,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "username is already claimed")
		}
		return nil, status.Errorf(codes.Internal, "set username: %v", err)
	}
	record, err := s.store.GetUsernameByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "username not found")
		}
		return nil, status.Errorf(codes.Internal, "set username: %v", err)
	}
	return &connectionsv1.SetUsernameResponse{
		UsernameRecord: usernameToProto(record),
	}, nil
}

// GetUsername fetches one canonical username by user ID.
func (s *Service) GetUsername(ctx context.Context, in *connectionsv1.GetUsernameRequest) (*connectionsv1.GetUsernameResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get username request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}
	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	record, err := s.store.GetUsernameByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "username not found")
		}
		return nil, status.Errorf(codes.Internal, "get username: %v", err)
	}
	return &connectionsv1.GetUsernameResponse{
		UsernameRecord: usernameToProto(record),
	}, nil
}

// LookupUsername resolves one canonical username to its owner user.
func (s *Service) LookupUsername(ctx context.Context, in *connectionsv1.LookupUsernameRequest) (*connectionsv1.LookupUsernameResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "lookup username request is required")
	}
	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "contact store is not configured")
	}
	canonicalUsername, err := usernameutil.Canonicalize(in.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "username is invalid: %v", err)
	}
	record, err := s.store.GetUsernameByUsername(ctx, canonicalUsername)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "username not found")
		}
		return nil, status.Errorf(codes.Internal, "lookup username: %v", err)
	}
	return &connectionsv1.LookupUsernameResponse{
		UsernameRecord: usernameToProto(record),
	}, nil
}

func contactToProto(contact storage.Contact) *connectionsv1.Contact {
	return &connectionsv1.Contact{
		OwnerUserId:   contact.OwnerUserID,
		ContactUserId: contact.ContactUserID,
		CreatedAt:     timestamppb.New(contact.CreatedAt),
		UpdatedAt:     timestamppb.New(contact.UpdatedAt),
	}
}

func usernameToProto(username storage.UsernameRecord) *connectionsv1.UsernameRecord {
	return &connectionsv1.UsernameRecord{
		UserId:    username.UserID,
		Username:  username.Username,
		CreatedAt: timestamppb.New(username.CreatedAt),
		UpdatedAt: timestamppb.New(username.UpdatedAt),
	}
}

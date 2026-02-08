package auth

import (
	"context"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/auth/user"
	apperrors "github.com/louisbranch/fracturing.space/internal/errors"
	"github.com/louisbranch/fracturing.space/internal/id"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListUsersPageSize = 10
	maxListUsersPageSize     = 50
)

// AuthService implements the auth.v1.AuthService gRPC API.
type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	store       storage.UserStore
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewAuthService creates an AuthService with default dependencies.
func NewAuthService(store storage.UserStore) *AuthService {
	return &AuthService{
		store:       store,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// CreateUser creates a new user record.
func (s *AuthService) CreateUser(ctx context.Context, in *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create user request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}

	created, err := user.CreateUser(user.CreateUserInput{DisplayName: in.GetDisplayName()}, s.clock, s.idGenerator)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if err := s.store.PutUser(ctx, created); err != nil {
		return nil, status.Errorf(codes.Internal, "put user: %v", err)
	}

	return &authv1.CreateUserResponse{User: userToProto(created)}, nil
}

// GetUser returns a user record by ID.
func (s *AuthService) GetUser(ctx context.Context, in *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get user request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	found, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &authv1.GetUserResponse{User: userToProto(found)}, nil
}

// ListUsers returns a page of user records.
func (s *AuthService) ListUsers(ctx context.Context, in *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list users request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListUsersPageSize
	}
	if pageSize > maxListUsersPageSize {
		pageSize = maxListUsersPageSize
	}

	page, err := s.store.ListUsers(ctx, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list users: %v", err)
	}

	response := &authv1.ListUsersResponse{NextPageToken: page.NextPageToken}
	if len(page.Users) == 0 {
		return response, nil
	}

	response.Users = make([]*authv1.User, 0, len(page.Users))
	for _, u := range page.Users {
		response.Users = append(response.Users, userToProto(u))
	}

	return response, nil
}

func userToProto(u user.User) *authv1.User {
	return &authv1.User{
		Id:          u.ID,
		DisplayName: u.DisplayName,
		CreatedAt:   timestamppb.New(u.CreatedAt),
		UpdatedAt:   timestamppb.New(u.UpdatedAt),
	}
}

// handleDomainError converts domain errors to gRPC status using the structured error system.
func handleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
}

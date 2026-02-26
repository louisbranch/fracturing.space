package auth

import (
	"context"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateWebSession creates a durable authenticated web session for a user.
func (s *AuthService) CreateWebSession(ctx context.Context, in *authv1.CreateWebSessionRequest) (*authv1.CreateWebSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create web session request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}
	if s.webSessionStore == nil {
		return nil, status.Error(codes.Internal, "web session store is not configured")
	}
	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	found, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	id, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate web session id: %v", err)
	}
	ttl := time.Duration(in.GetTtlSeconds()) * time.Second
	if ttl <= 0 {
		ttl = defaultWebSessionTTL
	}
	now := s.clock().UTC()
	session := storage.WebSession{ID: id, UserID: userID, CreatedAt: now, ExpiresAt: now.Add(ttl)}
	if err := s.webSessionStore.PutWebSession(ctx, session); err != nil {
		return nil, status.Errorf(codes.Internal, "put web session: %v", err)
	}
	return &authv1.CreateWebSessionResponse{Session: webSessionToProto(session), User: userToProto(found)}, nil
}

// GetWebSession resolves an authenticated web session by ID.
func (s *AuthService) GetWebSession(ctx context.Context, in *authv1.GetWebSessionRequest) (*authv1.GetWebSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get web session request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}
	if s.webSessionStore == nil {
		return nil, status.Error(codes.Internal, "web session store is not configured")
	}
	id := strings.TrimSpace(in.GetSessionId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	session, err := s.webSessionStore.GetWebSession(ctx, id)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, status.Error(codes.NotFound, "web session not found")
		}
		return nil, status.Errorf(codes.Internal, "get web session: %v", err)
	}
	now := s.clock().UTC()
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return nil, status.Error(codes.NotFound, "web session not found")
	}
	found, err := s.store.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	return &authv1.GetWebSessionResponse{Session: webSessionToProto(session), User: userToProto(found)}, nil
}

// RevokeWebSession revokes a durable authenticated web session by ID.
func (s *AuthService) RevokeWebSession(ctx context.Context, in *authv1.RevokeWebSessionRequest) (*authv1.RevokeWebSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke web session request is required")
	}
	if s.webSessionStore == nil {
		return nil, status.Error(codes.Internal, "web session store is not configured")
	}
	id := strings.TrimSpace(in.GetSessionId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}
	if err := s.webSessionStore.RevokeWebSession(ctx, id, s.clock().UTC()); err != nil {
		if err == storage.ErrNotFound {
			return &authv1.RevokeWebSessionResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "revoke web session: %v", err)
	}
	return &authv1.RevokeWebSessionResponse{}, nil
}

func webSessionToProto(session storage.WebSession) *authv1.WebSession {
	out := &authv1.WebSession{
		Id:        session.ID,
		UserId:    session.UserID,
		CreatedAt: timestamppb.New(session.CreatedAt),
		ExpiresAt: timestamppb.New(session.ExpiresAt),
	}
	if session.RevokedAt != nil {
		out.RevokedAt = timestamppb.New(*session.RevokedAt)
	}
	return out
}

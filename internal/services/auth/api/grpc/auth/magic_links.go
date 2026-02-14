package auth

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *AuthService) GenerateMagicLink(ctx context.Context, in *authv1.GenerateMagicLinkRequest) (*authv1.GenerateMagicLinkResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "generate magic link request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}
	if s.emailStore == nil {
		return nil, status.Error(codes.Internal, "email store is not configured")
	}
	if s.magicLinkStore == nil {
		return nil, status.Error(codes.Internal, "magic link store is not configured")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	requestedEmail := strings.TrimSpace(in.GetEmail())
	if requestedEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	parsed, err := mail.ParseAddress(requestedEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "email is invalid")
	}
	email := strings.ToLower(parsed.Address)

	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if existing, err := s.emailStore.GetUserEmailByEmail(ctx, email); err == nil {
		if existing.UserID != user.ID {
			return nil, status.Error(codes.AlreadyExists, "email already in use")
		}
	} else if err == storage.ErrNotFound {
		emailID, err := id.NewID()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generate email id: %v", err)
		}
		now := s.clock().UTC()
		if err := s.emailStore.PutUserEmail(ctx, storage.UserEmail{
			ID:        emailID,
			UserID:    user.ID,
			Email:     email,
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "store user email: %v", err)
		}
	} else {
		return nil, status.Errorf(codes.Internal, "get user email: %v", err)
	}

	linkToken, err := id.NewID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate magic link token: %v", err)
	}
	now := s.clock().UTC()
	expiresAt := now.Add(s.magicLinkConfig.TTL)
	pendingID := strings.TrimSpace(in.GetPendingId())
	if err := s.magicLinkStore.PutMagicLink(ctx, storage.MagicLink{
		Token:     linkToken,
		UserID:    user.ID,
		Email:     email,
		PendingID: pendingID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "store magic link: %v", err)
	}

	magicURL, err := buildMagicLinkURL(s.magicLinkConfig.BaseURL, linkToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build magic link url: %v", err)
	}

	return &authv1.GenerateMagicLinkResponse{
		MagicLinkUrl: magicURL,
		ExpiresAt:    timestamppb.New(expiresAt),
	}, nil
}

func (s *AuthService) ConsumeMagicLink(ctx context.Context, in *authv1.ConsumeMagicLinkRequest) (*authv1.ConsumeMagicLinkResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "consume magic link request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}
	if s.emailStore == nil {
		return nil, status.Error(codes.Internal, "email store is not configured")
	}
	if s.magicLinkStore == nil {
		return nil, status.Error(codes.Internal, "magic link store is not configured")
	}

	token := strings.TrimSpace(in.GetToken())
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	link, err := s.magicLinkStore.GetMagicLink(ctx, token)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, status.Error(codes.NotFound, "magic link not found")
		}
		return nil, status.Errorf(codes.Internal, "load magic link: %v", err)
	}

	now := s.clock().UTC()
	if link.UsedAt != nil {
		return nil, status.Error(codes.InvalidArgument, "magic link already used")
	}
	if now.After(link.ExpiresAt) {
		return nil, status.Error(codes.InvalidArgument, "magic link expired")
	}

	if err := s.magicLinkStore.MarkMagicLinkUsed(ctx, token, now); err != nil {
		return nil, status.Errorf(codes.Internal, "mark magic link used: %v", err)
	}
	if err := s.emailStore.VerifyUserEmail(ctx, link.UserID, link.Email, now); err != nil {
		return nil, status.Errorf(codes.Internal, "verify user email: %v", err)
	}

	if link.PendingID != "" {
		if err := s.attachPendingAuthorization(ctx, link.PendingID, link.UserID); err != nil {
			return nil, err
		}
	}

	user, err := s.store.GetUser(ctx, link.UserID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &authv1.ConsumeMagicLinkResponse{
		User:      userToProto(user),
		PendingId: link.PendingID,
	}, nil
}

func (s *AuthService) ListUserEmails(ctx context.Context, in *authv1.ListUserEmailsRequest) (*authv1.ListUserEmailsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list user emails request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "user store is not configured")
	}
	if s.emailStore == nil {
		return nil, status.Error(codes.Internal, "email store is not configured")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	if _, err := s.store.GetUser(ctx, userID); err != nil {
		return nil, handleDomainError(err)
	}

	rows, err := s.emailStore.ListUserEmailsByUser(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list user emails: %v", err)
	}
	items := make([]*authv1.UserEmail, 0, len(rows))
	for _, row := range rows {
		items = append(items, emailToProto(row))
	}
	return &authv1.ListUserEmailsResponse{Emails: items}, nil
}

func buildMagicLinkURL(base string, token string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", fmt.Errorf("base url is required")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func emailToProto(email storage.UserEmail) *authv1.UserEmail {
	result := &authv1.UserEmail{
		Email:     email.Email,
		CreatedAt: timestamppb.New(email.CreatedAt),
		UpdatedAt: timestamppb.New(email.UpdatedAt),
	}
	if email.VerifiedAt != nil {
		result.VerifiedAt = timestamppb.New(*email.VerifiedAt)
	}
	return result
}

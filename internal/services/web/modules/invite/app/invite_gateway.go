package app

import "context"

// Gateway loads and mutates public invite workflows.
type Gateway interface {
	GetPublicInvite(context.Context, string) (PublicInvite, error)
	AcceptInvite(context.Context, string, PublicInvite) error
	DeclineInvite(context.Context, string, string) error
}

// Service exposes invite landing workflows used by transport handlers.
type Service interface {
	LoadInvite(context.Context, string, string) (InvitePage, error)
	AcceptInvite(context.Context, string, string) (InviteMutationResult, error)
	DeclineInvite(context.Context, string, string) (InviteMutationResult, error)
}

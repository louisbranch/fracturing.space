package users

import (
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

func buildUserRows(users []*authv1.User) []templates.UserRow {
	rows := make([]templates.UserRow, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		rows = append(rows, templates.UserRow{
			ID:        user.GetId(),
			Email:     user.GetEmail(),
			CreatedAt: eventview.FormatTimestamp(user.GetCreatedAt()),
			UpdatedAt: eventview.FormatTimestamp(user.GetUpdatedAt()),
		})
	}
	return rows
}

func buildUserDetail(user *authv1.User) *templates.UserDetail {
	if user == nil {
		return nil
	}
	return &templates.UserDetail{
		ID:        user.GetId(),
		Email:     user.GetEmail(),
		CreatedAt: eventview.FormatTimestamp(user.GetCreatedAt()),
		UpdatedAt: eventview.FormatTimestamp(user.GetUpdatedAt()),
	}
}

func buildUserEmailRows(emails []*authv1.UserEmail, loc *message.Printer) []templates.UserEmailRow {
	rows := make([]templates.UserEmailRow, 0, len(emails))
	for _, email := range emails {
		if email == nil {
			continue
		}
		verified := "-"
		if email.GetVerifiedAt() != nil {
			verified = eventview.FormatTimestamp(email.GetVerifiedAt())
		}
		rows = append(rows, templates.UserEmailRow{
			Email:      email.GetEmail(),
			VerifiedAt: verified,
			CreatedAt:  eventview.FormatTimestamp(email.GetCreatedAt()),
			UpdatedAt:  eventview.FormatTimestamp(email.GetUpdatedAt()),
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return rows
}

func formatInviteStatus(status statev1.InviteStatus, loc *message.Printer) (string, string) {
	switch status {
	case statev1.InviteStatus_PENDING:
		return loc.Sprintf("label.invite_pending"), "warning"
	case statev1.InviteStatus_CLAIMED:
		return loc.Sprintf("label.invite_claimed"), "success"
	case statev1.InviteStatus_REVOKED:
		return loc.Sprintf("label.invite_revoked"), "error"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

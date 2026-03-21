package users

import (
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
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
		username := user.GetUsername()
		if username == "" {
			username = user.GetId()
		}
		rows = append(rows, templates.UserRow{
			ID:        user.GetId(),
			Username:  username,
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
	username := user.GetUsername()
	if username == "" {
		username = user.GetId()
	}
	return &templates.UserDetail{
		ID:        user.GetId(),
		Username:  username,
		CreatedAt: eventview.FormatTimestamp(user.GetCreatedAt()),
		UpdatedAt: eventview.FormatTimestamp(user.GetUpdatedAt()),
	}
}

func formatInviteStatus(status invitev1.InviteStatus, loc *message.Printer) (string, string) {
	switch status {
	case invitev1.InviteStatus_PENDING:
		return loc.Sprintf("label.invite_pending"), "warning"
	case invitev1.InviteStatus_CLAIMED:
		return loc.Sprintf("label.invite_claimed"), "success"
	case invitev1.InviteStatus_REVOKED:
		return loc.Sprintf("label.invite_revoked"), "error"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

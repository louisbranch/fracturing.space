package templates

// UsersPageView provides data for the users page.
type UsersPageView struct {
	Message       string
	Impersonation *ImpersonationView
}

// UserFormCardView provides data for a reusable user form card.
type UserFormCardView struct {
	Title       string
	Action      string
	Method      string
	FieldLabel  string
	FieldName   string
	FieldValue  string
	FieldType   string
	Placeholder string
	Required    bool
	ButtonLabel string
}

// UserDetailPageView provides data for the single user detail page.
type UserDetailPageView struct {
	Message            string
	Detail             *UserDetail
	Impersonation      *ImpersonationView
	MagicLinkURL       string
	MagicLinkEmail     string
	MagicLinkExpiresAt string
}

// UserRow represents a row in the users table.
type UserRow struct {
	ID          string
	DisplayName string
	CreatedAt   string
	UpdatedAt   string
}

// UserDetail represents a single user detail view.
type UserDetail struct {
	ID                    string
	DisplayName           string
	CreatedAt             string
	UpdatedAt             string
	Emails                []UserEmailRow
	PendingInvites        []InviteRow
	PendingInvitesMessage string
}

// UserEmailRow represents an email entry for a user.
type UserEmailRow struct {
	Email      string
	VerifiedAt string
	CreatedAt  string
	UpdatedAt  string
}

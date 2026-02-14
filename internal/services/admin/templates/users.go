package templates

// UsersPageView provides data for the users page.
type UsersPageView struct {
	Message       string
	Detail        *UserDetail
	Impersonation *ImpersonationView
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
	PendingInvites        []InviteRow
	PendingInvitesMessage string
}

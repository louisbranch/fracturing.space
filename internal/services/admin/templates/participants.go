package templates

// ParticipantRow represents a single row in the participants table.
type ParticipantRow struct {
	ID                string
	DisplayName       string
	Role              string
	RoleVariant       string
	Access            string
	AccessVariant     string
	Controller        string
	ControllerVariant string
	CreatedDate       string
}

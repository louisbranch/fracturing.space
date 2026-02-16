package invite

// Status identifies the invite status label.
type Status string

const (
	StatusUnspecified Status = ""
	StatusPending     Status = statusPending
	StatusClaimed     Status = statusClaimed
	StatusRevoked     Status = statusRevoked
)

// NormalizeStatus parses a status label into a canonical value.
func NormalizeStatus(value string) (Status, bool) {
	if normalized, ok := normalizeStatusLabel(value); ok {
		return Status(normalized), true
	}
	return StatusUnspecified, false
}

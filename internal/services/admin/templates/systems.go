// File systems.go defines view data for systems templates.
package templates

// SystemRow holds formatted system data for display.
type SystemRow struct {
	Name                string
	Version             string
	ImplementationStage string
	OperationalStatus   string
	AccessLevel         string
	IsDefault           bool
	DetailURL           string
}

// SystemDetail holds formatted system data for the detail page.
type SystemDetail struct {
	ID                  string
	Name                string
	Version             string
	ImplementationStage string
	OperationalStatus   string
	AccessLevel         string
	IsDefault           bool
}

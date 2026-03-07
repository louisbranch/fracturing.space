// File status.go defines view data for status templates.
package templates

// StatusCapabilityRow holds formatted capability data for display.
type StatusCapabilityRow struct {
	Service         string
	Capability      string
	ReportedStatus  string
	EffectiveStatus string
	StatusVariant   string
	HasOverride     bool
	OverrideDetail  string
}

// StatusServiceGroup holds capabilities grouped by service.
type StatusServiceGroup struct {
	Service         string
	AggregateStatus string
	StatusVariant   string
	Capabilities    []StatusCapabilityRow
	HasOverrides    bool
}

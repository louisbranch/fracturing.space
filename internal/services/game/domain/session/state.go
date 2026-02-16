package session

// State captures session facts derived from domain events.
type State struct {
	Started              bool
	Ended                bool
	SessionID            string
	Name                 string
	GateOpen             bool
	GateID               string
	SpotlightType        string
	SpotlightCharacterID string
}

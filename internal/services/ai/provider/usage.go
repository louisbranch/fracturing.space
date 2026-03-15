package provider

// Usage records provider-reported token consumption for one response.
type Usage struct {
	InputTokens     int32
	OutputTokens    int32
	ReasoningTokens int32
	TotalTokens     int32
}

// IsZero reports whether the provider returned no usage accounting.
func (u Usage) IsZero() bool {
	return u.InputTokens == 0 &&
		u.OutputTokens == 0 &&
		u.ReasoningTokens == 0 &&
		u.TotalTokens == 0
}

// Add combines two usage samples.
func (u Usage) Add(other Usage) Usage {
	return Usage{
		InputTokens:     u.InputTokens + other.InputTokens,
		OutputTokens:    u.OutputTokens + other.OutputTokens,
		ReasoningTokens: u.ReasoningTokens + other.ReasoningTokens,
		TotalTokens:     u.TotalTokens + other.TotalTokens,
	}
}

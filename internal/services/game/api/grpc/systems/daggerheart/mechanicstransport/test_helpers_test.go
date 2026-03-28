package mechanicstransport

func newTestHandler(seed int64) *Handler {
	return NewHandler(func() (int64, error) { return seed, nil })
}

func intPointer(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func stringPointer(value string) *string {
	return &value
}

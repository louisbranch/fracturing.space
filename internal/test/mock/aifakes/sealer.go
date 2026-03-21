package aifakes

// Sealer is an in-memory secret sealer fake for AI service tests.
type Sealer struct {
	SealErr error
	OpenErr error
}

// Seal returns a deterministic encrypted value unless configured to fail.
func (f *Sealer) Seal(value string) (string, error) {
	if f.SealErr != nil {
		return "", f.SealErr
	}
	return "enc:" + value, nil
}

// Open returns the plaintext value unless configured to fail.
func (f *Sealer) Open(sealed string) (string, error) {
	if f.OpenErr != nil {
		return "", f.OpenErr
	}
	const prefix = "enc:"
	if len(sealed) >= len(prefix) && sealed[:len(prefix)] == prefix {
		return sealed[len(prefix):], nil
	}
	return sealed, nil
}

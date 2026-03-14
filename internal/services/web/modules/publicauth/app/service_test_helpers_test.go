package app

// newService keeps a combined publicauth fixture seam available only to
// package tests. Production wiring should stay on the owned page/session/
// passkey/recovery constructors.
func newService(gateway Gateway, authBaseURL string) service {
	return newServiceState(gateway, authBaseURL)
}

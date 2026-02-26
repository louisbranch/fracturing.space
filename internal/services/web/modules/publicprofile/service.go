package publicprofile

type service struct{}

func newService() service {
	return service{}
}

func (service) body() string {
	return "web public profile"
}

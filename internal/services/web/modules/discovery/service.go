package discovery

type service struct{}

func newService() service {
	return service{}
}

func (service) body() string {
	return "web discovery"
}

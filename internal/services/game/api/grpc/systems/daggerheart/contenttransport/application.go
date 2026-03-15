package contenttransport

type contentApplication struct {
	handler *Handler
}

func newContentApplication(handler *Handler) contentApplication {
	return contentApplication{handler: handler}
}

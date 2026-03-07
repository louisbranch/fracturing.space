package game

// snapshotApplication coordinates snapshot transport use-cases across focused
// state patch/update helper files.
type snapshotApplication struct {
	stores Stores
}

func newSnapshotApplication(service *SnapshotService) snapshotApplication {
	return snapshotApplication{stores: service.stores}
}

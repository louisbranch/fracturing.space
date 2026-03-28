package modules

// NewDependencies returns module dependency defaults with shared runtime
// configuration applied.
func NewDependencies(assetBaseURL string) Dependencies {
	return Dependencies{AssetBaseURL: assetBaseURL}
}

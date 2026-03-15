package web

import grpc "google.golang.org/grpc"

// DependencyBinder maps one connected backend dependency into the assembled
// web runtime bundle.
type DependencyBinder func(*DependencyBundle, *grpc.ClientConn)

// StartupDependencyDescriptor describes one backend dependency the web service
// can bind during startup. Command-layer startup policy consumes this table
// instead of owning binder selection itself.
type StartupDependencyDescriptor struct {
	Name string
	Bind DependencyBinder
}

const (
	// DependencyNameAuth identifies the auth backend dependency.
	DependencyNameAuth = "auth"
	// DependencyNameSocial identifies the social backend dependency.
	DependencyNameSocial = "social"
	// DependencyNameGame identifies the game backend dependency.
	DependencyNameGame = "game"
	// DependencyNameAI identifies the AI backend dependency.
	DependencyNameAI = "ai"
	// DependencyNameDiscovery identifies the discovery backend dependency.
	DependencyNameDiscovery = "discovery"
	// DependencyNameUserHub identifies the userhub backend dependency.
	DependencyNameUserHub = "userhub"
	// DependencyNameNotifications identifies the notifications backend dependency.
	DependencyNameNotifications = "notifications"
	// DependencyNameStatus identifies the status backend dependency.
	DependencyNameStatus = "status"
)

var startupDependencyDescriptors = []StartupDependencyDescriptor{
	{Name: DependencyNameAuth, Bind: BindAuthDependency},
	{Name: DependencyNameSocial, Bind: BindSocialDependency},
	{Name: DependencyNameGame, Bind: BindGameDependency},
	{Name: DependencyNameAI, Bind: BindAIDependency},
	{Name: DependencyNameDiscovery, Bind: BindDiscoveryDependency},
	{Name: DependencyNameUserHub, Bind: BindUserHubDependency},
	{Name: DependencyNameNotifications, Bind: BindNotificationsDependency},
	{Name: DependencyNameStatus, Bind: BindStatusDependency},
}

// StartupDependencyDescriptors returns the stable startup dependency descriptor
// table consumed by command-layer bootstrap policy.
func StartupDependencyDescriptors() []StartupDependencyDescriptor {
	descriptors := make([]StartupDependencyDescriptor, len(startupDependencyDescriptors))
	copy(descriptors, startupDependencyDescriptors)
	return descriptors
}

// LookupStartupDependencyDescriptor returns the service-owned descriptor for
// one backend dependency.
func LookupStartupDependencyDescriptor(name string) (StartupDependencyDescriptor, bool) {
	for _, descriptor := range startupDependencyDescriptors {
		if descriptor.Name == name {
			return descriptor, true
		}
	}
	return StartupDependencyDescriptor{}, false
}

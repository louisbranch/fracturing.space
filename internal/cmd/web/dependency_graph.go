package web

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/web"
)

// closableManagedConn is the shutdown contract used by web startup wiring.
type closableManagedConn interface {
	Close() error
}

// managedConns captures the connection slice contract used during dependency
// bootstrap so runtime assembly can own shutdown in one place.
type managedConns []*platformgrpc.ManagedConn

// managedConnMode maps the web startup policy to the underlying managed-conn
// behavior used during bootstrap.
func managedConnMode(policy web.StartupDependencyPolicy) platformgrpc.ManagedConnMode {
	if policy == web.StartupDependencyRequired {
		return platformgrpc.ModeRequired
	}
	return platformgrpc.ModeOptional
}

// managedConnFactory builds one managed backend connection during startup.
type managedConnFactory func(context.Context, platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error)

// dependencyRequirement describes one startup dependency and its wiring step.
type dependencyRequirement struct {
	name       string
	address    string
	policy     web.StartupDependencyPolicy
	capability string
	surfaces   []string
	setInput   web.DependencyBinder
	onConnect  dependencyConnHook
}

// dependencyConnHook performs optional post-connect setup for one dependency.
type dependencyConnHook func(context.Context, *platformgrpc.ManagedConn)

// dependencyAddressResolver maps one command config to the backend address for
// a service-owned startup dependency descriptor.
type dependencyAddressResolver func(Config) string

// dependencyAddressBinding binds a startup dependency name to the command config
// field and service default.
type dependencyAddressBinding struct {
	descriptor web.StartupDependencyDescriptor
	address    func(*Config) *string
}

// resolve reads and normalizes the configured gRPC address from the startup
// binding.
func (b dependencyAddressBinding) resolve(cfg Config) string {
	if b.address == nil {
		return ""
	}
	return strings.TrimSpace(*b.address(&cfg))
}

// applyDependencyAddressDefaults fills unset dependency addresses from stable
// service defaults where a resolver is known.
func applyDependencyAddressDefaults(cfg *Config) {
	for _, descriptor := range web.StartupDependencyDescriptors() {
		binding, ok := dependencyAddressBindingForName(descriptor.Name)
		if !ok {
			continue
		}
		field := binding.address(cfg)
		*field = serviceaddr.OrDefaultGRPCAddr(*field, binding.descriptor.DefaultGRPCService)
	}
}

// dependencyAddressBindingNames returns startup dependency names in the canonical
// descriptor order, skipping unknown entries.
func dependencyAddressBindingNames() []string {
	descriptors := web.StartupDependencyDescriptors()
	names := make([]string, 0, len(descriptors))
	seen := map[string]struct{}{}
	for _, descriptor := range descriptors {
		if descriptor.Name == "" {
			continue
		}
		if _, ok := dependencyAddressBindingForName(descriptor.Name); !ok {
			continue
		}
		if _, duplicate := seen[descriptor.Name]; duplicate {
			continue
		}
		seen[descriptor.Name] = struct{}{}
		names = append(names, descriptor.Name)
	}
	return names
}

// applyDependencyAddressFlags wires startup dependency flags from the same canonical
// binding map used by resolver/default generation.
func applyDependencyAddressFlags(fs *flag.FlagSet, cfg *Config) {
	if fs == nil || cfg == nil {
		return
	}
	for _, name := range dependencyAddressBindingNames() {
		binding, ok := dependencyAddressBindingForName(name)
		if !ok {
			continue
		}
		field := binding.address(cfg)
		fs.StringVar(field, dependencyAddressFlagName(name), *field, dependencyAddressFlagUsage(name))
	}
}

// dependencyAddressFlagName maps one canonical dependency name to CLI flag key
// naming.
func dependencyAddressFlagName(name string) string {
	return fmt.Sprintf("%s-addr", name)
}

// dependencyAddressFlagUsage returns the usage text for one dependency flag.
func dependencyAddressFlagUsage(name string) string {
	if name == "" {
		return "dependency gRPC address"
	}
	return fmt.Sprintf("%s gRPC dependency address", name)
}

// dependencyAddressBindingForName resolves one binding from a canonical dependency
// name.
func dependencyAddressBindingForName(name string) (dependencyAddressBinding, bool) {
	descriptor, ok := web.LookupStartupDependencyDescriptor(name)
	if !ok {
		return dependencyAddressBinding{}, false
	}
	return dependencyAddressBindingForDescriptor(descriptor)
}

// dependencyAddressBindingForDescriptor resolves command-layer config-field
// ownership for one service-owned startup dependency descriptor.
func dependencyAddressBindingForDescriptor(descriptor web.StartupDependencyDescriptor) (dependencyAddressBinding, bool) {
	binding := dependencyAddressBinding{
		descriptor: descriptor,
		address:    dependencyAddressField(descriptor.Name),
	}
	return binding, binding.address != nil && binding.descriptor.DefaultGRPCService != ""
}

// dependencyAddressField maps a service-owned startup dependency name to the
// command config field that stores its gRPC address.
func dependencyAddressField(name string) func(*Config) *string {
	switch name {
	case web.DependencyNameAuth:
		return func(cfg *Config) *string { return &cfg.AuthAddr }
	case web.DependencyNameSocial:
		return func(cfg *Config) *string { return &cfg.SocialAddr }
	case web.DependencyNameGame:
		return func(cfg *Config) *string { return &cfg.GameAddr }
	case web.DependencyNameAI:
		return func(cfg *Config) *string { return &cfg.AIAddr }
	case web.DependencyNameDiscovery:
		return func(cfg *Config) *string { return &cfg.DiscoveryAddr }
	case web.DependencyNameUserHub:
		return func(cfg *Config) *string { return &cfg.UserHubAddr }
	case web.DependencyNameNotifications:
		return func(cfg *Config) *string { return &cfg.NotificationsAddr }
	case web.DependencyNameStatus:
		return func(cfg *Config) *string { return &cfg.StatusAddr }
	default:
		return nil
	}
}

// DependencyAddressResolverContractError reports mismatches between service-owned
// dependency descriptors and command-layer address resolver configuration.
type DependencyAddressResolverContractError struct {
	Missing []string
	Extra   []string
}

// Error returns a stable summary of descriptor resolver mismatch state.
func (e DependencyAddressResolverContractError) Error() string {
	if len(e.Missing) == 0 && len(e.Extra) == 0 {
		return "startup dependency address resolver contract is complete"
	}
	return fmt.Sprintf("startup dependency address resolver contract mismatch: missing=%v extras=%v", e.Missing, e.Extra)
}

// MissingRequiredStartupDependencyAddressesError reports required startup
// dependencies that do not have configured addresses.
type MissingRequiredStartupDependencyAddressesError struct {
	Missing []string
}

// Error returns a stable summary of missing required dependency addresses.
func (e MissingRequiredStartupDependencyAddressesError) Error() string {
	return fmt.Sprintf("required startup dependency addresses are missing: %v", e.Missing)
}

// dependencyAddressResolverDefaults returns a stable snapshot of resolver wiring so
// callers cannot mutate global state through shared map references.
func dependencyAddressResolverDefaults() map[string]dependencyAddressResolver {
	descriptors := web.StartupDependencyDescriptors()
	resolvers := make(map[string]dependencyAddressResolver, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Name == "" {
			continue
		}
		resolve := dependencyAddressResolverForName(descriptor.Name)
		if resolve == nil {
			continue
		}
		resolvers[descriptor.Name] = resolve
	}
	return resolvers
}

// dependencyAddressResolverForName maps service-owned dependency names to command config
// address resolvers so resolver wiring stays explicit and centralized.
func dependencyAddressResolverForName(name string) dependencyAddressResolver {
	binding, ok := dependencyAddressBindingForName(name)
	if !ok {
		return nil
	}
	return binding.resolve
}

// dependencyRequirements returns startup requirements in stable dependency
// order and fails fast when command-layer address wiring drifts from the
// service-owned descriptor table.
func dependencyRequirements(cfg Config, reporter *platformstatus.Reporter) ([]dependencyRequirement, error) {
	return dependencyRequirementsWithResolvers(cfg, reporter, dependencyAddressResolverDefaults())
}

// dependencyRequirementsWithResolvers builds startup requirements from the active
// dependency descriptors and resolver map.
func dependencyRequirementsWithResolvers(
	cfg Config,
	reporter *platformstatus.Reporter,
	resolvers map[string]dependencyAddressResolver,
) ([]dependencyRequirement, error) {
	if err := validateDependencyAddressResolversCoverageWithResolvers(resolvers); err != nil {
		return nil, err
	}

	descriptors := web.StartupDependencyDescriptors()
	requirements := make([]dependencyRequirement, 0, len(descriptors))
	missingRequiredAddresses := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		address, err := dependencyAddress(cfg, descriptor.Name, resolvers)
		if err != nil {
			return nil, err
		}
		if descriptor.Policy == web.StartupDependencyRequired && strings.TrimSpace(address) == "" {
			missingRequiredAddresses = append(missingRequiredAddresses, descriptor.Name)
		}
		requirements = append(requirements, dependencyRequirement{
			name:       descriptor.Name,
			address:    address,
			policy:     descriptor.Policy,
			capability: descriptor.Capability,
			surfaces:   append([]string(nil), descriptor.Surfaces...),
			setInput:   descriptor.Bind,
			onConnect:  dependencyOnConnect(descriptor.Name, reporter),
		})
	}

	if len(missingRequiredAddresses) > 0 {
		sort.Strings(missingRequiredAddresses)
		return nil, MissingRequiredStartupDependencyAddressesError{
			Missing: missingRequiredAddresses,
		}
	}

	return requirements, nil
}

// dependencyAddress resolves the configured backend address for one
// service-owned startup dependency descriptor.
func dependencyAddress(cfg Config, name string, resolvers map[string]dependencyAddressResolver) (string, error) {
	resolve, ok := resolvers[name]
	if !ok {
		return "", fmt.Errorf("web startup dependency %q is missing a command-layer address resolver", name)
	}
	return resolve(cfg), nil
}

// validateDependencyAddressResolversCoverage checks resolver coverage against the
// startup descriptor table in the active process.
func validateDependencyAddressResolversCoverage() error {
	return validateDependencyAddressResolversCoverageWithResolvers(dependencyAddressResolverDefaults())
}

// validateDependencyAddressResolversCoverageWithResolvers asserts that resolver
// coverage mirrors service-owned startup descriptors.
func validateDependencyAddressResolversCoverageWithResolvers(resolvers map[string]dependencyAddressResolver) error {
	descriptors := web.StartupDependencyDescriptors()
	descriptorByName := make(map[string]struct{}, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Name == "" {
			continue
		}
		descriptorByName[descriptor.Name] = struct{}{}
	}

	missing := make([]string, 0, len(descriptorByName))
	for _, descriptor := range descriptors {
		if descriptor.Name == "" {
			continue
		}
		if _, ok := resolvers[descriptor.Name]; !ok {
			missing = append(missing, descriptor.Name)
		}
	}

	extra := make([]string, 0, len(resolvers))
	for name := range resolvers {
		if name == "" {
			continue
		}
		if _, ok := descriptorByName[name]; !ok {
			extra = append(extra, name)
		}
	}

	if len(missing) == 0 && len(extra) == 0 {
		return nil
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return DependencyAddressResolverContractError{
		Missing: missing,
		Extra:   extra,
	}
}

// dependencyOnConnect returns any late-binding hook that should run after one
// dependency connects, keeping those side effects out of the descriptor table.
func dependencyOnConnect(name string, reporter *platformstatus.Reporter) dependencyConnHook {
	if name == web.DependencyNameStatus {
		return bindStatusReporter(reporter)
	}
	return nil
}

// bindStatusReporter late-binds the status reporter client once the connection
// becomes healthy.
func bindStatusReporter(reporter *platformstatus.Reporter) dependencyConnHook {
	if reporter == nil {
		return nil
	}
	return func(ctx context.Context, mc *platformgrpc.ManagedConn) {
		if mc == nil {
			return
		}
		client := statusv1.NewStatusServiceClient(mc.Conn())
		go func() {
			if mc.WaitReady(ctx) == nil {
				reporter.SetClient(client)
			}
		}()
	}
}

// bootstrapDependencies creates ManagedConns for each requirement and wires
// gRPC clients into the dependency bundle. Required deps block until healthy;
// optional deps return immediately.
func bootstrapDependencies(
	ctx context.Context,
	requirements []dependencyRequirement,
	assetBaseURL string,
	reporter *platformstatus.Reporter,
	logger *slog.Logger,
	newConn managedConnFactory,
) (web.DependencyBundle, managedConns, error) {
	bundle := web.NewDependencyBundle(assetBaseURL)
	var conns managedConns
	logger = web.LoggerOrDefault(logger)
	if newConn == nil {
		newConn = platformgrpc.NewManagedConn
	}

	logf := func(format string, args ...any) {
		logger.Info(fmt.Sprintf(format, args...))
	}

	for _, dep := range requirements {
		if strings.TrimSpace(dep.address) == "" {
			continue
		}
		mc, err := newConn(ctx, platformgrpc.ManagedConnConfig{
			Name:             dep.name,
			Addr:             dep.address,
			Mode:             managedConnMode(dep.policy),
			Logf:             logf,
			StatusReporter:   reporter,
			StatusCapability: dep.capability,
		})
		if err != nil {
			closeManagedConns(conns, logger)
			return web.DependencyBundle{}, nil, fmt.Errorf("dependency %s: %w", dep.name, err)
		}
		conns = append(conns, mc)
		if dep.setInput != nil {
			dep.setInput(&bundle, mc.Conn())
		}
		if dep.onConnect != nil {
			dep.onConnect(ctx, mc)
		}
	}

	return bundle, conns, nil
}

// closeManagedConns closes all ManagedConn instances.
func closeManagedConns(conns managedConns, logger *slog.Logger) {
	for _, mc := range conns {
		closeManagedConn(mc, "dependency", logger)
	}
}

// closeManagedConn nil-safely closes a ManagedConn with error logging.
func closeManagedConn(mc closableManagedConn, name string, logger *slog.Logger) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		web.LoggerOrDefault(logger).Error("close web managed conn", "name", name, "error", err)
	}
}

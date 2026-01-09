# HyperDHCP - AI Coding Agent Instructions

## Project Overview
HyperDHCP is a Kubernetes operator that manages DHCP servers with KubeVirt integration. It consists of two main components:
- **Operator (manager)**: Kubernetes controller that watches `Server` custom resources and creates DHCP server deployments
- **DHCP Server (hyperdhcpd)**: CoreDHCP-based DHCP service with custom plugins for KubeVirt VM networking

## Architecture Patterns

### Dual Binary Architecture
- `cmd/manager/` - Operator binary using controller-runtime
- `cmd/server/` - DHCP server binary using CoreDHCP framework
- Run with `./hyperdhcp manager` or `./hyperdhcp server`

### Custom Resource Definition (CRD)
- `Server` CRD in `api/v1beta1/` defines DHCP configuration
- Controller in `internal/controller/` reconciles Server resources to Deployments
- Each Server creates: Deployment + ConfigMap + PVC + ServiceAccount

### DHCP Plugin System
- Extends CoreDHCP with custom plugins in `internal/dhcp/plugins/`
- `kubevirt` plugin: Discovers VM network interfaces via KubeVirt API
- `leasedb` plugin: SQLite-based lease storage with bitmap allocation
- Plugins registered in `internal/dhcp/server.go`

### Network Integration
- Uses Multus CNI for network attachment (see `k8s.v1.cni.cncf.io/networks` annotation)
- DHCP servers bind to specific network interfaces via `NetworkAttachment` spec
- Privileged containers required for raw socket access

## Development Workflows

### Building
```bash
make build                    # Build operator binary
make build-hyperdhcp-release  # Build DHCP server binary
make build-hyperdhcp-image    # Build DHCP container image
make build-image             # Build operator container image
```

### Testing
```bash
make test                    # Run unit tests with envtest
make manifests               # Generate CRDs/RBAC before testing
```

### Local Development
```bash
make run                     # Run operator locally against cluster
make deploy                  # Deploy operator to cluster
make install                 # Install CRDs to cluster
```

### Code Generation
```bash
make generate                # Generate deepcopy methods
make manifests               # Generate CRDs/webhooks/RBAC
controller-gen crd:trivialVersions=true paths="./..." output:crd:artifacts:config=config/crd/bases
```

## Code Patterns & Conventions

### Controller Implementation
- Use `CreateOrUpdateWithRetries` for idempotent resource management
- Set controller ownership with `ctrl.SetControllerReference`
- Follow kubebuilder patterns: `// +kubebuilder:rbac:` comments for RBAC

### DHCP Plugin Development
- Implement `plugins.Plugin` interface with `Setup4` function
- Return `handler.Handler4` for packet processing
- Use CoreDHCP logger: `logger.GetLogger("plugins/name")`

### Configuration Management
- Viper-based config with `.hyperdhcp.yaml` default
- Environment variables override file config
- CLI flags via cobra framework

### Error Handling
- Use controller-runtime logging: `log.FromContext(ctx)`
- Return errors from reconcile loops for requeue
- Use `client.IgnoreNotFound` for deletion handling

### Testing
- Use Ginkgo/Gomega for controller tests
- envtest for API server integration testing
- Table-driven tests for plugin logic

## Key Files & Directories

- `api/v1beta1/server_types.go` - Server CRD specification
- `internal/controller/server_controller.go` - Main reconciliation logic
- `internal/dhcp/server.go` - DHCP server setup and plugin loading
- `internal/dhcp/plugins/` - Custom DHCP plugins (kubevirt, leasedb)
- `config/` - Kustomize manifests for deployment
- `Makefile` - Build and development targets

## Integration Points

### KubeVirt API
- Watches `VirtualMachineInstance` resources
- Discovers network interfaces for IP allocation
- Client in `internal/dhcp/plugins/kubevirt/client/`

### Kubernetes Networking
- Multus network attachment definitions
- Service accounts for network access
- NetworkAttachment custom resources

### Storage
- PVC for lease database persistence
- ConfigMap for DHCP server configuration
- SQLite backend for lease records

## Common Tasks

### Adding New DHCP Plugin
1. Create `internal/dhcp/plugins/newplugin/plugin.go`
2. Implement `plugins.Plugin` with `Setup4` function
3. Register in `internal/dhcp/server.go` plugins slice
4. Add tests in `plugin_test.go`

### Modifying Server CRD
1. Edit `api/v1beta1/server_types.go`
2. Run `make generate` to update deepcopy methods
3. Run `make manifests` to update CRDs
4. Update controller logic in `server_controller.go`

### Adding Controller Logic
1. Modify `Reconcile` method in controller
2. Add RBAC annotations for new API calls
3. Run `make manifests` to update RBAC
4. Add unit tests for new logic
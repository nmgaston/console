# Redfish Plugin for Device Management Toolkit

This directory contains the Redfish v1 API implementation for the Device Management Toolkit Console, implemented as a plugin using the DMT plugin architecture.

## Overview

This implementation provides a minimal Redfish service compliant with DMTF Redfish specifications. It integrates with DMT's shared infrastructure (configuration, logging, database, authentication) and provides the basic endpoints required for a Redfish-compliant management service.

## Architecture

The Redfish implementation follows DMT's plugin architecture pattern:

```text
redfish/
├── plugin.go                          # Plugin entry point and registration
├── pkg/plugin.go                       # Public plugin interface
├── internal/                          # Internal implementation
│   ├── controller/http/v1/            # HTTP handlers and routes
│   │   ├── generated/                 # Auto-generated OpenAPI code
│   │   └── handler/                   # Custom handlers and middleware
│   ├── entity/v1/                     # Domain entities and types
│   └── usecase/                       # Business logic and repository interfaces
└── openapi/                           # OpenAPI specifications and tooling
    ├── dmtf/                          # DMTF Redfish schema files
    ├── merged/                        # Merged OpenAPI specifications
    └── infra/                         # Build tooling and Makefile
```### Plugin Integration

The Redfish plugin integrates with DMT using the plugin interface:

- **Plugin Registration**: Automatically registered in `internal/controller/http/router.go`
- **Shared Infrastructure**: Uses DMT's config, logging, database, and HTTP router
- **Route Registration**: Routes are registered directly on the main router at `/redfish/v1/*`
- **Authentication**: Can be configured to use DMT's JWT authentication or run unprotected

## Endpoints Implemented

### Service Root

- `GET /redfish/v1/` - Returns the service root with links to major collections

### Systems

- `GET /redfish/v1/Systems` - Computer Systems collection
- `GET /redfish/v1/Systems/{systemId}` - Individual Computer System
- `POST /redfish/v1/Systems/{systemId}/Actions/ComputerSystem.Reset` - System reset action

### Chassis

- `GET /redfish/v1/Chassis` - Chassis collection
- `GET /redfish/v1/Chassis/{chassisId}` - Individual Chassis

### Managers

- `GET /redfish/v1/Managers` - Managers collection
- `GET /redfish/v1/Managers/{managerId}` - Individual Manager

### Metadata

- `GET /redfish/v1/$metadata` - OData metadata document

## Configuration

The plugin can be configured through environment variables or YAML:

```yaml
# In config.yml (if plugin configuration is added)
redfish:
  enabled: true
  auth_required: false
  base_url: "/redfish/v1"
```

Environment variables:

- `REDFISH_ENABLED=true` - Enable/disable the plugin
- `REDFISH_AUTH_REQUIRED=false` - Require JWT authentication
- `REDFISH_BASE_URL=/redfish/v1` - Base URL for Redfish endpoints

## Testing

### Start the Application

```bash
# Build the application
go build ./cmd/app/

# Start with default config
./app

# Start with custom config
./app -config ~/configs/console-config.yml
```

### Test Endpoints

```bash
# Test service root
curl -k https://localhost:8181/redfish/v1/ | jq .

# Test systems collection
curl -k https://localhost:8181/redfish/v1/Systems | jq .

# Test individual system
curl -k https://localhost:8181/redfish/v1/Systems/System1 | jq .

# Test chassis collection
curl -k https://localhost:8181/redfish/v1/Chassis | jq .

# Test managers collection
curl -k https://localhost:8181/redfish/v1/Managers | jq .

# Test system reset (POST action)
curl -k -X POST https://localhost:8181/redfish/v1/Systems/System1/Actions/ComputerSystem.Reset \
  -H "Content-Type: application/json" \
  -d '{"ResetType": "ForceRestart"}' | jq .
```

## OpenAPI Code Generation

The Redfish implementation uses OpenAPI-first development with automated code generation.

### Makefile Usage

Navigate to the `redfish/openapi/infra/` directory to use the Makefile:

```bash
cd redfish/openapi/infra/
```

#### Available Targets

**Help and Information:**

```bash
make help                    # Display all available targets
make rf-check-tools         # Check if required tools are installed
```

**Dependency Management:**

```bash
make rf-deps                # Install all required dependencies
make rf-install-missing-tools # Install only missing tools
```

**Code Generation:**

```bash
make rf-merge               # Merge YAML files into single OpenAPI spec
make rf-generate            # Generate Go server code from OpenAPI spec
make rf-validate            # Validate the merged OpenAPI specification
make rf-all                 # Run all steps: merge, generate, and validate
```

**Cleanup:**

```bash
make rf-clean               # Clean all generated files
```

### Development Workflow

1. **Check dependencies:**

   ```bash
   make rf-check-tools
   ```

2. **Install missing tools (if needed):**

   ```bash
   make rf-deps
   ```

3. **Modify OpenAPI specifications:**
   - Edit files in `redfish/openapi/dmtf/` directory
   - Add new YAML schema files as needed

4. **Regenerate code:**

   ```bash
   make rf-all
   ```

5. **Implement handlers:**
   - Update `redfish/internal/controller/http/v1/handler/routes.go`
   - Add business logic in `redfish/internal/usecase/`

6. **Test changes:**

   ```bash
   go build ./cmd/app/
   ./app -config ~/configs/console-config.yml
   ```

### Required Tools

The Makefile will check for and install these tools:

**Required:**

- `python3` - For merging OpenAPI specifications
- `go` - Go compiler and runtime
- `oapi-codegen` - OpenAPI code generator for Go
- `PyYAML` - Python YAML processing library

**Optional:**

- `jq` - JSON processor for testing
- `curl` - HTTP client for testing
- `swagger-cli` - OpenAPI validation tool

### Generated Files

The Makefile generates these files:

- `redfish/openapi/merged/redfish-openapi.yaml` - Merged OpenAPI specification
- `redfish/internal/controller/http/v1/generated/types.gen.go` - Generated types
- `redfish/internal/controller/http/v1/generated/server.gen.go` - Generated server interface
- `redfish/internal/controller/http/v1/generated/spec.gen.go` - Generated specification

## Integration with DMT

### Shared Infrastructure

The plugin leverages DMT's shared infrastructure:

- **Configuration Management**: YAML-based config with environment variable overrides
- **Logging**: Structured logging with configurable levels
- **Database Access**: Repository pattern with transaction management
- **HTTP Router**: Gin-based router with middleware pipeline
- **Authentication**: JWT-based authentication (optional)

### Plugin Lifecycle

1. **Registration**: Plugin is registered in the main router
2. **Initialization**: Plugin initializes with shared DMT context
3. **Route Registration**: Routes are registered on the main HTTP router
4. **Runtime**: Plugin handles Redfish requests using shared infrastructure
5. **Shutdown**: Clean shutdown when application terminates

## DMTF Redfish Compliance

This implementation is based on:

- **DMTF Redfish Specification v1.19.0**
- **OpenAPI schemas from DMTF Redfish-Publications repository (2025.3)**
- **Standard Redfish endpoints**: Service Root, Systems, Chassis, and Manager resources
- **Redfish error responses**: Compliant error handling with proper message formats

## Future Enhancements

Planned enhancements include:

- **Additional Resources**: Processors, Memory, Storage, NetworkInterfaces
- **Authentication Integration**: Full JWT middleware integration
- **WebSocket Support**: Server-Sent Events for subscriptions
- **Database Integration**: Persistent storage for device information
- **Task Service**: Long-running operation tracking
- **Full CRUD Operations**: POST, PATCH, DELETE support
- **Action Endpoints**: Additional system management actions
- **Event Subscriptions**: Redfish event notification support

## Troubleshooting

### Common Issues

**Plugin not loading:**

- Check that `REDFISH_ENABLED=true` is set
- Verify plugin registration in router.go

**Routes returning 404:**

- Ensure routes are registered on main router, not router groups
- Check that BaseURL is empty string in RegisterHandlersWithOptions

**Build failures:**

- Run `make rf-check-tools` to verify dependencies
- Regenerate code with `make rf-all`

**OpenAPI generation errors:**

- Ensure you're on Ubuntu (required for Makefile)
- Install missing dependencies with `make rf-deps`
- Check YAML syntax in dmtf/ directory

### Debug Mode

Enable debug logging by setting environment variables:

```bash
export GIN_MODE=debug
export LOG_LEVEL=debug
```

This will provide detailed request/response logging and OpenAPI documentation generation.

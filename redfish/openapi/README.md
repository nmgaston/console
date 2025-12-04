# Redfish OpenAPI Code Generation

The Redfish implementation uses OpenAPI-first development with automated code generation managed through a Makefile located in `infra/`.

## System Requirements

- Ubuntu 22.04 or later
- Python 3.x with PyYAML
- Go 1.24+

## Makefile Usage

**Navigation:**
All commands must be run from the `infra/` directory:

```bash
cd infra/
```

### Help Commands

```bash
make help                    # Display all available targets with descriptions
make rf-check-tools          # Check if all required tools are available
```

### Dependency Management

```bash
make rf-deps                  # Install dependencies and required tools for Redfish API
make rf-install-missing-tools # Install missing tools automatically
```

### OpenAPI Processing

```bash
make rf-merge                # Merge YAML files into single OpenAPI specification
make rf-validate             # Validate OpenAPI specification
```

### Code Generation

```bash
make rf-generate             # Generate Go server code from merged OpenAPI spec
make rf-auth                 # Add Basic Auth to OpenAPI spec and regenerate Go code
```

### Complete Workflow

```bash
make rf-all                  # Run all Redfish API generation steps
```

### Cleanup

```bash
make rf-clean                # Clean generated files
```

## Development Workflow

1. **Check Prerequisites:**

   ```bash
   make rf-check-tools
   ```

2. **Install Missing Tools:**

   ```bash
   make rf-deps
   ```

3. **Modify OpenAPI Specifications (optional - needed when adding or modifying the dmtf openapi specs):**

   - Add or modify YAML files in the `dmtf/` directory for new Redfish resources according to what would be implemented
   - Update `dmtf/openapi-reduced.yaml` to reference any new schema files
   - Ensure proper DMTF Redfish schema format compliance

4. **Complete Generation Process:**

   ```bash
   make rf-all
   ```

   This runs the full pipeline: merge YAML files → validate → generate Go code

5. **Individual Steps (if needed):**

   ```bash
   make rf-merge      # Just merge YAML files
   make rf-generate   # Just generate Go code
   make rf-validate   # Just validate OpenAPI spec
   ```

## Integration Testing

The Redfish API implementation includes comprehensive integration tests using Newman/Postman:

```bash
make rf-integration-test
```

This runs automated tests against a mock Redfish server that validate:
- All OpenAPI endpoints (100% coverage)
- Power control actions (On, ForceOff, ForceRestart, GracefulShutdown, PowerCycle)
- Authentication and authorization
- Error handling (invalid requests, malformed JSON, etc.)
- DMTF Redfish standard compliance

**Test Requirements:**
- `newman` - Newman CLI for running Postman collections
- `jq` - JSON processor (optional, for enhanced reporting)

The test runner automatically:
- Builds the application
- Starts a mock Redfish server on port 8181
- Runs 22 requests with 66 assertions
- Displays formatted test results
- Cleans up resources on completion

Server logs are saved to `/tmp/redfish_test_server.log` and displayed only on test failures.

## Tool Requirements

The Makefile automatically checks for and can install:

**Required:**

- `python3` - For YAML processing and merging
- `go` - Go compiler and runtime
- `oapi-codegen` - OpenAPI code generator for Go
- `PyYAML` - Python YAML processing library
- `swagger-cli` - OpenAPI validation tool

## File Processing

**Input:** YAML files from `dmtf/` directory

**Output:**

- `merged/redfish-openapi.yaml` - Merged OpenAPI specification
- `../internal/controller/http/v1/generated/types.gen.go` - Generated Go types
- `../internal/controller/http/v1/generated/server.gen.go` - Generated server interfaces
- `../internal/controller/http/v1/generated/spec.gen.go` - Generated specification embedding

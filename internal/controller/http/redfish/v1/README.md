# Redfish v1 Implementation

This directory contains the Redfish v1 API implementation for the Device Management Toolkit Console.

## Overview

This implementation provides a minimal Redfish service compliant with DMTF Redfish specifications. It includes the basic endpoints required to start building a Redfish-compliant management service.

## Files

- `openapi.yaml` - OpenAPI 3.0 specification defining the Redfish v1 endpoints
- `generated.go` - Auto-generated Go types and server interface using oapi-codegen
- `server.go` - Implementation of the Redfish server interface with mock data
- `routes.go` - Route setup and registration with Gin router

## Endpoints Implemented

### Service Root

- `GET /redfish/v1/` - Returns the service root with links to major collections

### Systems

- `GET /redfish/v1/Systems` - Computer Systems collection
- `GET /redfish/v1/Systems/{systemId}` - Individual Computer System

### Chassis

- `GET /redfish/v1/Chassis` - Chassis collection
- `GET /redfish/v1/Chassis/{chassisId}` - Individual Chassis

### Managers

- `GET /redfish/v1/Managers` - Managers collection  
- `GET /redfish/v1/Managers/{managerId}` - Individual Manager

## Testing

Start the server and test the endpoints:

```bash
# Start the server
go run cmd/app/main.go

# Test service root
curl http://localhost:8181/redfish/v1/ | jq .

# Test systems collection
curl http://localhost:8181/redfish/v1/Systems | jq .

# Test individual system
curl http://localhost:8181/redfish/v1/Systems/1 | jq .
```

## Development Workflow

To extend the API:

1. Modify `openapi.yaml` to add new endpoints and schemas
2. Regenerate the Go code: `oapi-codegen -generate types,gin -package v1 openapi.yaml > generated.go`
3. Implement the new handlers in `server.go`
4. Test the new endpoints

## Architecture

The implementation follows these patterns:

- **OpenAPI-First**: All APIs are defined in `openapi.yaml` first
- **Code Generation**: Server stubs and types are generated from the OpenAPI spec
- **Interface Implementation**: Business logic is implemented by satisfying the generated `ServerInterface`
- **Gin Integration**: Uses Gin HTTP framework for routing and middleware

## Future Enhancements

This is a minimal starting point. Future enhancements could include:

- Additional Redfish resources (Processors, Memory, Storage, etc.)
- Authentication and authorization
- WebSocket support for Server-Sent Events
- Database integration for persistent data
- Task service for long-running operations
- Full CRUD operations (POST, PATCH, DELETE)
- Action endpoints for system management (power, reset, etc.)

## DMTF Redfish Compliance

This implementation is based on:

- DMTF Redfish Specification v1.15.0
- OpenAPI schemas from DMTF Redfish-Publications repository (2025.3)
- Redfish Service Root, Systems, Chassis, and Manager resources

For full DMTF compliance, additional resources and endpoints would need to be implemented according to the Redfish specification.

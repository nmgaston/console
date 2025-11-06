# Redfish API Integration Tests

This directory contains integration tests for the Console Redfish v1 API implementation.

## Test Structure

### Postman/Newman Tests
Located in `collections/console_redfish_apis.postman_collection.json`

**Test Coverage:**
1. **Service Root** - Validates Redfish service root endpoint
2. **Systems Collection** - Tests ComputerSystem collection
3. **ComputerSystem Resource** - Tests individual system retrieval
4. **Reset Action Success** - Tests valid power state changes
5. **Error Handling:**
   - 400 Bad Request (invalid ResetType, missing properties)
   - 401 Unauthorized (missing/invalid JWT)
   - 404 Not Found (non-existent system)
   - 405 Method Not Allowed (wrong HTTP method)

### Running Tests Locally

**IMPORTANT**: Authentication must be disabled for local testing:
```bash
export AUTH_DISABLED=true
go run cmd/app/main.go
```

See [AUTH_CONFIG_FIX.md](AUTH_CONFIG_FIX.md) for details on auth configuration.

#### With Docker Compose:
```bash
# Start the console service
docker compose up -d --build

# Run Redfish API tests
docker run --network=host \
  -v $(pwd)/integration-test/collections:/collections \
  -v $(pwd)/integration-test/results:/results \
  postman/newman:5.3-alpine run \
  /collections/console_redfish_apis.postman_collection.json \
  -e /collections/console_environment.postman_environment.json \
  --insecure \
  --reporters cli,json,junit \
  --reporter-json-export /results/redfish_results.json \
  --reporter-junit-export /results/redfish_results_junit.xml
```

#### With Postman Desktop:
1. Import `console_redfish_apis.postman_collection.json`
2. Import `console_environment.postman_environment.json`
3. Update environment variables for your local setup
4. Run the collection

### Environment Variables

The tests use these variables from `console_environment.postman_environment.json`:
- `base_url` - Console API base URL (e.g., `http://localhost:8181`)
- `jwt_token` - JWT bearer token for authentication

## CI/CD Integration

Tests run automatically in GitHub Actions on:
- Push to `main` branch
- Pull requests to `main`
- Manual workflow dispatch

See `.github/workflows/api-test.yml` for configuration.

## Adding New Tests

### Postman Collection
1. Open the collection in Postman
2. Add new request to appropriate folder
3. Add test scripts in the "Tests" tab:
   ```javascript
   pm.test("Status code is 200", function () {
       pm.response.to.have.status(200);
   });
   
   pm.test("Response has required property", function () {
       var jsonData = pm.response.json();
       pm.expect(jsonData).to.have.property("propertyName");
   });
   ```
4. Export and save to `collections/`

### Test Scenarios to Add

**Power State Transitions:**
- [ ] Test 409 Conflict (invalid state transitions)
- [ ] Test ForceOff, ForceRestart, PowerCycle actions
- [ ] Test power state validation logic

**Service Availability:**
- [ ] Test 503 Service Unavailable (backend connection issues)
- [ ] Verify Retry-After header

**Authorization:**
- [ ] Test 403 Forbidden (insufficient privileges)
- [ ] Test role-based access control

**Additional Resources:**
- [ ] Chassis collection and resources
- [ ] Managers collection and resources
- [ ] Test OData metadata endpoint

## Go-Based Integration Tests (Future)

The `main_test.go` file contains commented-out Go-based tests using the `go-hit` library. This approach offers:
- Type-safe request/response handling
- Better integration with Go tooling
- Easier debugging and maintenance

To enable:
1. Uncomment tests in `main_test.go`
2. Add Redfish-specific test functions
3. Run with `go test ./integration-test/...`

## Redfish Compliance

Tests validate compliance with:
- **Redfish Specification v1.11.0**
- **DMTF CIM Schema** (power state constants)
- **OData v4.0** (headers and response format)

All error responses must include:
- `@Message.ExtendedInfo` array
- `MessageId` (e.g., `Base.1.11.0.PropertyMissing`)
- `Message`, `Severity`, `Resolution` fields

## Troubleshooting

**Tests failing with 401?**
- Ensure JWT token is valid and not expired
- Check `console_environment.postman_environment.json` has correct token

**Tests failing with connection errors?**
- Verify console service is running: `docker compose ps`
- Check service logs: `docker compose logs console`
- Ensure ports are accessible (default: 8181)

**Tests passing locally but failing in CI?**
- Check if service startup takes longer in CI (adjust sleep time)
- Verify environment variables in GitHub Actions
- Check Docker network configuration

## References

- [Redfish Specification](https://www.dmtf.org/standards/redfish)
- [Newman Documentation](https://learning.postman.com/docs/running-collections/using-newman-cli/command-line-integration-with-newman/)
- [Postman Test Scripts](https://learning.postman.com/docs/writing-scripts/test-scripts/)

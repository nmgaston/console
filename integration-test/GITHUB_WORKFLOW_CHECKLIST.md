# GitHub Workflow Readiness Checklist

## ✅ Ready for GitHub Actions

The Redfish API tests are now configured to run in GitHub Actions workflows.

## What Was Fixed

### 1. ✅ Added AUTH_DISABLED to .env.example

- Added `AUTH_DISABLED=false` (default for production)
- Workflow overrides this with `AUTH_DISABLED=true` for Redfish tests

### 2. ✅ Separated Test Execution Strategy

**MPS/RPS Tests** (existing):

- Run via Docker Compose (requires full stack)
- Uses authentication

**Redfish Tests** (new):

- Run via native Go (`go run cmd/app/main.go`)
- Uses `AUTH_DISABLED=true` environment variable
- More reliable (avoids Alpine CDN timeout issues)

### 3. ✅ Added Go Setup Action

- Installs Go 1.23
- Required for `go run` approach

### 4. ✅ Added Health Check

- Curls `/redfish/v1/` before running tests
- Fails fast with logs if service doesn't start
- Validates auth is actually disabled

### 5. ✅ Added Cleanup Steps

- Stops Docker Compose before starting Go service
- Kills Go service after tests complete
- Prevents port conflicts

### 6. ✅ Enhanced Error Reporting

- Dumps console.log on failure
- Shows Docker logs on failure
- Uploads test results even on failure (`if: always()`)

## Workflow Execution Flow

```
1. Checkout code
2. Install Go 1.23
3. Copy .env.example → .env (AUTH_DISABLED=false)
4. Start Docker Compose (for MPS/RPS)
5. Run MPS/RPS API tests
6. Stop Docker Compose
7. Start native Go service (with AUTH_DISABLED=true)
8. Health check Redfish endpoint
9. Run Redfish API tests
10. Stop Go service
11. Upload results & logs
```

## Expected Test Results

When the workflow runs:

✅ **MPS/RPS tests should PASS** (existing functionality)
✅ **Redfish tests should show 15/18 assertions passing**
⚠️ **3 assertions will FAIL** (Reset Action backend not implemented)

This is **expected behavior** - the failing tests are documented in `TEST_RESULTS_SUMMARY.md`.

## Monitoring the Workflow

### Where to Check

- GitHub Actions tab in repository
- Workflow: "Console API Tests"
- Triggered on: Push to `main`, PRs to `main`, manual dispatch

### What to Look For

**Success indicators:**

- ✅ "Start Console Service (Native Go for Redfish tests)" - shows health check passed
- ✅ "Run Redfish API Tests" - shows 15/18 assertions passing
- ⚠️ Exit code 1 is EXPECTED (due to 3 failing assertions)

**Failure indicators:**

- ❌ Health check fails - check console.log in artifacts
- ❌ Service won't start - check console.log dump
- ❌ All tests return 401 - AUTH_DISABLED not working

### Downloading Test Results

After workflow completes:

1. Go to workflow run page
2. Scroll to "Artifacts" section
3. Download `api-test-results.zip`
4. Contains:
   - `console_api_results.json` (MPS/RPS tests)
   - `console_api_results_junit.xml` (MPS/RPS JUnit format)
   - `console_redfish_api_results.json` (Redfish tests)
   - `console_redfish_api_results_junit.xml` (Redfish JUnit format)

## Known Issues in CI

### Expected Failures (3 assertions)

These will fail until Reset Action backend is implemented:

1. "Status code is 202 Accepted" - Returns 404
2. "Response has Location header" - No header
3. "Task response is valid" - Returns error instead of task

### If You See Different Failures

**401 Unauthorized errors:**

- Check AUTH_DISABLED environment variable
- Verify logs show "without authentication"
- Check console.log artifact

**Service won't start:**

- Check for port conflicts (unlikely in CI)
- Check Go version compatibility
- Review console.log artifact

**Newman/Docker issues:**

- Check network connectivity
- Verify Docker Hub access
- Check postman/newman:5.3-alpine image availability

## Next Steps

1. **Merge to main** - Workflow will run automatically
2. **Monitor first run** - Download artifacts and verify results
3. **Implement Reset Action backend** - All 18 assertions should pass
4. **Update this checklist** - Document any issues found in CI

## Local Testing Before Push

Test the exact workflow locally:

```bash
# Simulate GitHub Actions environment
export AUTH_DISABLED=true
go run cmd/app/main.go > /tmp/console.log 2>&1 &
PID=$!
sleep 5

# Health check
curl -f http://localhost:8181/redfish/v1/ || (cat /tmp/console.log && kill $PID && exit 1)

# Run tests
cd integration-test
docker run --rm --network host \
  -v "$(pwd)/collections:/etc/newman" \
  postman/newman:5.3-alpine \
  run console_redfish_apis.postman_collection.json \
  -e console_environment.postman_environment.json

# Cleanup
kill $PID
```

Expected: 15/18 assertions passing ✅

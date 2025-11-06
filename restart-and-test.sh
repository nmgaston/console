#!/bin/bash
set -e

echo "=== Stopping All Console Processes ==="
pkill -9 -f "go run cmd/app/main" 2>/dev/null || true
pkill -9 -f "/tmp/go-build" 2>/dev/null || true
lsof -ti:8181 2>/dev/null | xargs kill -9 2>/dev/null || true
sleep 3

echo "=== Verifying Port is Free ==="
if lsof -ti:8181 >/dev/null 2>&1; then
    echo "ERROR: Port 8181 is still in use!"
    lsof -ti:8181
    exit 1
fi
echo "✓ Port 8181 is free"

echo ""
echo "=== Checking Configuration ==="
echo "Config file auth settings:"
grep -A3 "^auth:" config/config.yml
echo ""
echo ".env AUTH_DISABLED setting:"
grep "AUTH_DISABLED" .env || echo "(not set in .env)"
echo ""

echo "=== Starting Console Service ==="
export AUTH_DISABLED=true
go run cmd/app/main.go > /tmp/console-final.log 2>&1 &
PID=$!
echo "Started with PID: $PID"
echo "Waiting 10 seconds for service to start..."
sleep 10

echo ""
echo "=== Checking Service Status ==="
if ! ps -p $PID > /dev/null; then
    echo "ERROR: Service failed to start!"
    echo "Last 20 lines of log:"
    tail -20 /tmp/console-final.log
    exit 1
fi
echo "✓ Service is running (PID: $PID)"

echo ""
echo "=== Testing Authentication ==="
echo "Attempting to access /redfish/v1/ ..."
RESPONSE=$(curl -s http://localhost:8181/redfish/v1/)
if echo "$RESPONSE" | grep -q '"@odata.id"'; then
    echo "✓ SUCCESS! Auth is disabled, got valid JSON response"
    echo "$RESPONSE" | jq '."@odata.id"' 2>/dev/null || echo "$RESPONSE"
elif echo "$RESPONSE" | grep -q "InsufficientPrivilege"; then
    echo "✗ FAILED! Still getting 401 Unauthorized"
    echo "Check logs: tail -50 /tmp/console-final.log"
    exit 1
else
    echo "? Unexpected response:"
    echo "$RESPONSE"
    exit 1
fi

echo ""
echo "=== Running Quick Tests ==="
./quick-test.sh

echo ""
echo "==================================="
echo "Service is ready! Run full tests with:"
echo "  docker run --network=host -v \$(pwd)/integration-test/collections:/collections postman/newman:5.3-alpine run /collections/console_redfish_apis.postman_collection.json -e /collections/console_environment.postman_environment.json --insecure"

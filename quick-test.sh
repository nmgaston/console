#!/bin/bash

echo "Quick Redfish API Test"
echo "======================"
echo ""

# Test 1: Service Root
echo "1. Testing Service Root (GET /redfish/v1/)..."
curl -s http://localhost:8181/redfish/v1/ | head -5
echo ""

# Test 2: Systems Collection  
echo "2. Testing Systems Collection (GET /redfish/v1/Systems)..."
curl -s http://localhost:8181/redfish/v1/Systems | head -5
echo ""

# Test 3: ComputerSystem
echo "3. Testing ComputerSystem (GET /redfish/v1/Systems/System1)..."
curl -s http://localhost:8181/redfish/v1/Systems/System1 | head -5
echo ""

# Test 4: Reset Action
echo "4. Testing Reset Action (POST with ResetType=On)..."
curl -s -X POST http://localhost:8181/redfish/v1/Systems/System1/Actions/ComputerSystem.Reset \
  -H "Content-Type: application/json" \
  -d '{"ResetType":"On"}' | head -5
echo ""

echo "======================"
echo "If you see JSON with @odata.id fields, auth is disabled and tests should work!"
echo "If you see 401 errors, auth is still enabled."

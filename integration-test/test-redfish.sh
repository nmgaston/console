#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Redfish API Integration Test Runner${NC}"
echo "========================================"
echo ""

# Check if console service is running
echo -e "${YELLOW}1. Checking if console service is running...${NC}"
if curl -s http://localhost:8181/healthz > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Console service is running${NC}"
else
    echo -e "${RED}✗ Console service is NOT running${NC}"
    echo -e "${YELLOW}Starting service with docker compose...${NC}"
    docker compose up -d --build
    echo -e "${YELLOW}Waiting 10 seconds for service to start...${NC}"
    sleep 10
    
    if curl -s http://localhost:8181/healthz > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Console service started successfully${NC}"
    else
        echo -e "${RED}✗ Failed to start console service${NC}"
        echo "Check logs with: docker compose logs console"
        exit 1
    fi
fi
echo ""

# Check if JWT authentication is disabled
echo -e "${YELLOW}2. Checking authentication configuration...${NC}"
if grep -q "disabled: true" config/config.yml 2>/dev/null; then
    echo -e "${GREEN}✓ JWT authentication is disabled (auth.disabled: true)${NC}"
    echo -e "  Tests will run without authentication"
else
    echo -e "${YELLOW}⚠ JWT authentication may be enabled${NC}"
    echo -e "  If tests fail with 401, either:"
    echo -e "  1. Set auth.disabled: true in config/config.yml and restart"
    echo -e "  2. Or get a JWT token and update console_environment.postman_environment.json"
    echo ""
    echo -e "${YELLOW}To get a JWT token (if auth is enabled):${NC}"
    echo '  curl -X POST http://localhost:8181/api/v1/authorize \'
    echo '    -H "Content-Type: application/json" \'
    echo '    -d '\''{"username":"admin","password":"your-password"}'\'''
fi
echo ""

# Run the Redfish API tests
echo -e "${YELLOW}3. Running Redfish API tests...${NC}"
echo "========================================"
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

# Check test results
if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo -e "Results saved to: integration-test/results/redfish_results.json"
else
    echo ""
    echo -e "${RED}✗ Some tests failed${NC}"
    echo -e "Check results in: integration-test/results/redfish_results.json"
    exit 1
fi

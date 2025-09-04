#!/bin/bash

# Script to run the real LLM integration tests
# This will use actual API calls to test tool integration

echo "========================================="
echo "Running Real LLM Integration Tests"
echo "========================================="
echo ""
echo "NOTE: This test requires:"
echo "1. Valid API credentials in config/config.yaml"
echo "2. Network access to the LLM API"
echo "3. Set SKIP_LLM_TEST=1 to skip these tests"
echo ""
echo "========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if we should skip
if [ "$SKIP_LLM_TEST" = "1" ]; then
    echo -e "${YELLOW}Skipping LLM tests (SKIP_LLM_TEST is set)${NC}"
    exit 0
fi

# Check if config file exists
if [ ! -f "../config/config.yaml" ] && [ ! -f "./config/config.yaml" ]; then
    echo -e "${RED}Error: config.yaml not found!${NC}"
    echo "Please create a config file with your API credentials"
    echo "Example location: ./config/config.yaml or ../config/config.yaml"
    exit 1
fi

echo "Starting tests..."
echo ""

# Run the simple test first
echo "1. Running simple tool call test..."
go test -v ./chat-new -run TestRealLLMSimpleToolCall -timeout 60s
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Simple tool call test passed${NC}"
else
    echo -e "${RED}✗ Simple tool call test failed${NC}"
fi

echo ""
echo "2. Running tool chaining test..."
go test -v ./chat-new -run TestRealLLMToolChaining -timeout 120s
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Tool chaining test passed${NC}"
else
    echo -e "${RED}✗ Tool chaining test failed${NC}"
fi

echo ""
echo "3. Running comprehensive all-tools test..."
echo "   (This may take a while as it tests all available tools)"
go test -v ./chat-new -run TestRealLLMWithAllTools -timeout 300s
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ All tools test passed${NC}"
else
    echo -e "${RED}✗ All tools test failed${NC}"
fi

echo ""
echo "========================================="
echo "LLM Integration Tests Complete"
echo "========================================="
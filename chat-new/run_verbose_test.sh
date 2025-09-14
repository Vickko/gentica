#!/bin/bash

# Verbose test runner - shows all LLM interactions and tool calls

echo "================================================="
echo "     LLM Tools Integration Test (Verbose)"
echo "================================================="
echo ""
echo "This will show:"
echo "  • All prompts sent to the LLM"
echo "  • Each tool call with arguments"
echo "  • Tool execution results"
echo "  • Final LLM responses"
echo "  • File verification results"
echo ""
echo "================================================="

# Colors for better readability
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to run test with nice formatting
run_test() {
    local test_name=$1
    local timeout=$2
    
    echo ""
    echo -e "${BLUE}=================================================${NC}"
    echo -e "${BLUE}Running: $test_name${NC}"
    echo -e "${BLUE}=================================================${NC}"
    
    go test -v ./chat-new -run "$test_name" -timeout "$timeout" 2>&1 | while IFS= read -r line; do
        # Highlight different types of output
        if [[ "$line" == *"=== Starting"* ]] || [[ "$line" == *"=== Summary"* ]]; then
            echo -e "${YELLOW}$line${NC}"
        elif [[ "$line" == *"✅"* ]]; then
            echo -e "${GREEN}$line${NC}"
        elif [[ "$line" == *"Tool called:"* ]] || [[ "$line" == *"Tool "*":"* ]]; then
            echo -e "${BLUE}$line${NC}"
        elif [[ "$line" == *"PASS"* ]]; then
            echo -e "${GREEN}$line${NC}"
        elif [[ "$line" == *"FAIL"* ]] || [[ "$line" == *"❌"* ]]; then
            echo -e "\033[0;31m$line${NC}"
        else
            echo "$line"
        fi
    done
}

# Ask user which test to run
echo "Which test would you like to run?"
echo "1) Simple Tool Call Test (quick)"
echo "2) Tool Chaining Test (medium)"
echo "3) All Tools Test (comprehensive)"
echo "4) Run all tests"
echo ""
read -p "Enter choice (1-4): " choice

case $choice in
    1)
        run_test "TestRealLLMSimpleToolCall" "60s"
        ;;
    2)
        run_test "TestRealLLMToolChaining" "120s"
        ;;
    3)
        run_test "TestRealLLMWithAllTools" "300s"
        ;;
    4)
        run_test "TestRealLLMSimpleToolCall" "60s"
        run_test "TestRealLLMToolChaining" "120s"
        run_test "TestRealLLMWithAllTools" "300s"
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}=================================================${NC}"
echo -e "${GREEN}         Test Execution Complete!${NC}"
echo -e "${GREEN}=================================================${NC}"
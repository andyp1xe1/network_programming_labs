#!/bin/bash

# KV Store Integration Test Script
# Tests leader-follower replication with write/delete/exists operations

set -e  # Exit on error

# Configuration
LEADER_URL="http://localhost:9000"
FOLLOWER_URLS=("http://localhost:9001" "http://localhost:9002" "http://localhost:9003" "http://localhost:9004" "http://localhost:9005")
TEST_ID="integration-test-$$"  # Unique test ID using PID

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Wait for services to be ready
wait_for_services() {
    log_info "Waiting for services to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$LEADER_URL/status" > /dev/null 2>&1; then
            log_success "Leader is ready"
            break
        fi
        attempt=$((attempt + 1))
        log_info "Waiting for leader... (attempt $attempt/$max_attempts)"
        sleep 2
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log_error "Leader failed to start within timeout"
        exit 1
    fi
    
    # Wait a bit more for followers
    sleep 3
    log_success "All services should be ready"
}

# Make HTTP request with proper error handling
make_request() {
    local url="$1"
    local data="$2"
    local expected_success="${3:-true}"
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$data" \
        "$url" 2>/dev/null)
    
    # Check if response is empty or not valid JSON
    if [ -z "$response" ] || ! echo "$response" | jq . >/dev/null 2>&1; then
        log_error "Invalid or empty response from $url"
        log_error "Response: '$response'"
        return 1
    fi
    
    local success
    success=$(echo "$response" | jq -r '.success // false')
    
    if [ "$success" != "$expected_success" ]; then
        log_error "Request failed. Expected success: $expected_success, got: $success"
        log_error "Response: $response"
        return 1
    fi
    
    echo "$response"
}

# Test write operation on leader and read from all followers
test_write_operation() {
    local key="$1"
    local value="$2"
    
    log_info "Testing write operation: key=$key, value=$value"
    
    # Write to leader
    local set_data='{"key": "'$key'", "value": "'$value'", "id": "'$TEST_ID'"}'
    log_info "Writing to leader..."
    make_request "$LEADER_URL/set" "$set_data" > /dev/null
    log_success "Write completed on leader"
    
    # Allow some time for replication
    sleep 2
    
    # Read from all followers
    local get_data='{"key": "'$key'"}'
    local success_count=0
    
    for i in "${!FOLLOWER_URLS[@]}"; do
        local follower_url="${FOLLOWER_URLS[$i]}"
        log_info "Reading from follower $((i+1))..."
        
        local response
        if response=$(make_request "$follower_url/get" "$get_data" 2>/dev/null); then
            local success
            success=$(echo "$response" | jq -r '.success // false' 2>/dev/null || echo "false")
            
            if [ "$success" = "true" ]; then
                local read_value
                read_value=$(echo "$response" | jq -r '.value // ""' 2>/dev/null || echo "")
                if [ "$read_value" = "$value" ]; then
                    log_success "Follower $((i+1)): Value matches ($read_value)"
                    success_count=$((success_count + 1))
                else
                    log_warning "Follower $((i+1)): Value mismatch. Expected: $value, Got: $read_value"
                fi
            else
                log_warning "Follower $((i+1)): Read failed - $response"
            fi
        else
            log_warning "Follower $((i+1)): Request failed or invalid response"
        fi
    done
    
    log_info "Replication success rate: $success_count/${#FOLLOWER_URLS[@]} followers"
    return 0
}

# Test delete operation on leader and check on all followers
test_delete_operation() {
    local key="$1"
    
    log_info "Testing delete operation: key=$key"
    
    # Delete on leader
    local delete_data='{"key": "'$key'", "id": "'$TEST_ID'"}'
    log_info "Deleting on leader..."
    make_request "$LEADER_URL/delete" "$delete_data" > /dev/null
    log_success "Delete completed on leader"
    
    # Allow some time for replication
    sleep 4
    
    # Check deletion on all followers using exists endpoint
    local exists_data='{"key": "'$key'"}'
    local success_count=0
    
    for i in "${!FOLLOWER_URLS[@]}"; do
        local follower_url="${FOLLOWER_URLS[$i]}"
        log_info "Checking deletion on follower $((i+1))..."
        
        local response
        response=$(make_request "$follower_url/exists" "$exists_data" 2>/dev/null || echo '{"success": false}')
        
        local success
        success=$(echo "$response" | jq -r '.success // false')
        
        if [ "$success" = "true" ]; then
            local exists
            exists=$(echo "$response" | jq -r '.exists')
            if [ "$exists" = "false" ]; then
                log_success "Follower $((i+1)): Key successfully deleted"
                success_count=$((success_count + 1))
            else
                log_warning "Follower $((i+1)): Key still exists after delete"
            fi
        else
            log_warning "Follower $((i+1)): Exists check failed"
        fi
    done
    
    log_info "Delete replication success rate: $success_count/${#FOLLOWER_URLS[@]} followers"
    return 0
}

# Test exists operation across all nodes
test_exists_operation() {
    local key="$1"
    local should_exist="$2"  # true or false
    
    log_info "Testing exists operation: key=$key (should_exist=$should_exist)"
    
    local exists_data='{"key": "'$key'"}'
    
    # Check on leader first
    log_info "Checking exists on leader..."
    local response
    response=$(make_request "$LEADER_URL/exists" "$exists_data")
    local exists
    exists=$(echo "$response" | jq -r '.exists // false')
    
    if [ "$exists" = "$should_exist" ]; then
        log_success "Leader: Exists check correct ($exists)"
    else
        log_warning "Leader: Exists check incorrect. Expected: $should_exist, Got: $exists"
    fi
    
    # Check on all followers
    local success_count=0
    for i in "${!FOLLOWER_URLS[@]}"; do
        local follower_url="${FOLLOWER_URLS[$i]}"
        log_info "Checking exists on follower $((i+1))..."
        
        response=$(make_request "$follower_url/exists" "$exists_data" 2>/dev/null || echo '{"success": false}')
        local success
        success=$(echo "$response" | jq -r '.success // false')
        
        if [ "$success" = "true" ]; then
            exists=$(echo "$response" | jq -r '.exists')
            if [ "$exists" = "$should_exist" ]; then
                log_success "Follower $((i+1)): Exists check correct ($exists)"
                success_count=$((success_count + 1))
            else
                log_warning "Follower $((i+1)): Exists check incorrect. Expected: $should_exist, Got: $exists"
            fi
        else
            log_warning "Follower $((i+1)): Exists check failed"
        fi
    done
    
    log_info "Exists check consistency: $success_count/${#FOLLOWER_URLS[@]} followers"
    return 0
}

# Main test execution
main() {
    echo "=================================================="
    echo "KV Store Integration Test"
    echo "Testing leader-follower replication"
    echo "=================================================="
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed. Please install jq."
        exit 1
    fi
    
    # Wait for services
    wait_for_services
    
    echo
    log_info "Starting integration tests..."
    echo
    
    # Test 1: Write and read operations
    echo "--- Test 1: Write Operations ---"
    test_write_operation "test_key_1" "test_value_1"
    test_write_operation "user:123" "john_doe"
    test_write_operation "config:timeout" "30s"
    echo
    
    # Test 2: Exists operations (keys should exist)
    echo "--- Test 2: Exists Operations (Positive) ---"
    test_exists_operation "test_key_1" "true"
    test_exists_operation "user:123" "true"
    test_exists_operation "config:timeout" "true"
    echo
    
    # Test 3: Delete operations
    echo "--- Test 3: Delete Operations ---"
    test_delete_operation "test_key_1"
    test_delete_operation "user:123"
    echo
    
    # Test 4: Exists operations (deleted keys should not exist)
    echo "--- Test 4: Exists Operations (Negative) ---"
    test_exists_operation "test_key_1" "false"
    test_exists_operation "user:123" "false"
    test_exists_operation "nonexistent_key" "false"
    echo
    
    # Test 5: Verify remaining key still exists
    echo "--- Test 5: Verify Remaining Data ---"
    test_exists_operation "config:timeout" "true"
    echo
    
    echo "=================================================="
    log_success "Integration test completed!"
    echo "Check the logs above for any warnings or errors."
    echo "=================================================="
}

# Execute main function
main "$@"
#!/bin/bash

# KV Store Versioning Integration Test Script
# Tests optimistic concurrency control, version tracking, and conflict resolution

set -e  # Exit on error

# Configuration
LEADER_URL="http://localhost:9000"
FOLLOWER_URLS=("http://localhost:9001" "http://localhost:9002" "http://localhost:9003" "http://localhost:9004" "http://localhost:9005")
TEST_ID="versioning-test-$$"  # Unique test ID using PID

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

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

log_test() {
    echo -e "${PURPLE}[TEST]${NC} $1"
    TESTS_RUN=$((TESTS_RUN + 1))
}

pass_test() {
    log_success "PASS: $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail_test() {
    log_error "FAIL: $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
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
        if [ "$expected_success" = "true" ]; then
            log_error "Request failed unexpectedly"
            log_error "Response: $response"
            return 1
        fi
    fi
    
    echo "$response"
}

# Extract version from response
get_version() {
    local response="$1"
    echo "$response" | jq -r '.metadata.version // 0' 2>/dev/null
}

# Extract value from response  
get_value() {
    local response="$1"
    echo "$response" | jq -r '.value // ""' 2>/dev/null
}

# Test basic versioning - set, get, update cycle
test_basic_versioning() {
    log_test "Basic Versioning Cycle"
    
    local key="version_test_1_${TEST_ID}"
    local value1="initial_value"
    local value2="updated_value"
    
    # Initial set (new key creation with expected_version 0)
    local set_data='{"key": "'$key'", "value": "'$value1'", "expected_version": 0, "id": "'$TEST_ID'"}'
    local response=$(make_request "$LEADER_URL/set" "$set_data")
    local version1=$(get_version "$response")
    
    if [ "$version1" = "1" ]; then
        pass_test "Initial set created version 1"
    else
        fail_test "Initial set should create version 1, got $version1"
        return 1
    fi
    
    # Get and verify version
    local get_data='{"key": "'$key'"}'
    response=$(make_request "$LEADER_URL/get" "$get_data")
    local get_version=$(get_version "$response")
    local get_value=$(get_value "$response")
    
    if [ "$get_version" = "1" ] && [ "$get_value" = "$value1" ]; then
        pass_test "Get returned correct version and value"
    else
        fail_test "Get failed: version=$get_version, value=$get_value"
        return 1
    fi
    
    # Update with correct expected version
    local update_data='{"key": "'$key'", "value": "'$value2'", "expected_version": 1, "id": "'$TEST_ID'"}'
    response=$(make_request "$LEADER_URL/set" "$update_data")
    local version2=$(get_version "$response")
    
    if [ "$version2" = "2" ]; then
        pass_test "Update incremented version to 2"
    else
        fail_test "Update should increment version to 2, got $version2"
        return 1
    fi
    
    # Verify update
    response=$(make_request "$LEADER_URL/get" "$get_data")
    get_version=$(get_version "$response")
    get_value=$(get_value "$response")
    
    if [ "$get_version" = "2" ] && [ "$get_value" = "$value2" ]; then
        pass_test "Updated value and version verified"
    else
        fail_test "Update verification failed: version=$get_version, value=$get_value"
        return 1
    fi
    
    log_success "Basic versioning cycle completed successfully"
}

# Test version conflict detection
test_version_conflicts() {
    log_test "Version Conflict Detection"
    
    local key="conflict_test_${TEST_ID}"
    local value1="value1"
    local value2="value2"
    local value3="value3"
    
    # Set initial value (new key creation with expected_version 0)
    local set_data='{"key": "'$key'", "value": "'$value1'", "expected_version": 0, "id": "'$TEST_ID'"}'
    local response=$(make_request "$LEADER_URL/set" "$set_data")
    local version=$(get_version "$response")
    
    if [ "$version" = "1" ]; then
        pass_test "Initial value set with version 1"
    else
        fail_test "Expected version 1, got $version"
        return 1
    fi
    
    # Try to update with wrong expected version (should fail)
    local wrong_update='{"key": "'$key'", "value": "'$value2'", "expected_version": 2, "id": "'$TEST_ID'"}'
    response=$(make_request "$LEADER_URL/set" "$wrong_update" "false")
    local success=$(echo "$response" | jq -r '.success')
    local error=$(echo "$response" | jq -r '.error // ""')
    
    if [ "$success" = "false" ] && [[ "$error" == *"version conflict"* ]]; then
        pass_test "Version conflict correctly detected and rejected"
    else
        fail_test "Version conflict should have been detected: success=$success, error=$error"
        return 1
    fi
    
    # Update with correct expected version (should succeed)
    local correct_update='{"key": "'$key'", "value": "'$value3'", "expected_version": 1, "id": "'$TEST_ID'"}'
    response=$(make_request "$LEADER_URL/set" "$correct_update")
    version=$(get_version "$response")
    
    if [ "$version" = "2" ]; then
        pass_test "Correct expected version allowed update"
    else
        fail_test "Update with correct expected version failed, got version $version"
        return 1
    fi
    
    log_success "Version conflict detection working correctly"
}

# Test concurrent update prevention
test_concurrent_updates() {
    log_test "Concurrent Update Prevention"
    
    local key="concurrent_test_${TEST_ID}"
    local initial_value="initial"
    local client1_value="client1_update"
    local client2_value="client2_update"
    
    # Set initial value (new key creation with expected_version 0)
    local set_data='{"key": "'$key'", "value": "'$initial_value'", "expected_version": 0, "id": "'$TEST_ID'"}'
    local response=$(make_request "$LEADER_URL/set" "$set_data")
    local version=$(get_version "$response")
    
    if [ "$version" = "1" ]; then
        pass_test "Initial value set for concurrent test"
    else
        fail_test "Failed to set initial value, got version $version"
        return 1
    fi
    
    # Simulate concurrent updates with same expected version
    local update1='{"key": "'$key'", "value": "'$client1_value'", "expected_version": 1, "id": "client1"}'
    local update2='{"key": "'$key'", "value": "'$client2_value'", "expected_version": 1, "id": "client2"}'
    
    # Run both updates in background using temporary files
    local tmp1="/tmp/response1_$$"
    local tmp2="/tmp/response2_$$"
    
    (make_request "$LEADER_URL/set" "$update1" "any" 2>/dev/null > "$tmp1") &
    local pid1=$!
    (make_request "$LEADER_URL/set" "$update2" "any" 2>/dev/null > "$tmp2") &
    local pid2=$!
    
    # Wait for both to complete
    wait $pid1
    wait $pid2
    
    # Get responses (one should succeed, one should fail)
    local response1=$(cat "$tmp1" 2>/dev/null)
    local response2=$(cat "$tmp2" 2>/dev/null)
    local success1=$(echo "$response1" | jq -r '.success // false' 2>/dev/null)
    local success2=$(echo "$response2" | jq -r '.success // false' 2>/dev/null)
    
    # Clean up temp files
    rm -f "$tmp1" "$tmp2" 2>/dev/null
    
    # Exactly one should succeed
    local success_count=0
    [ "$success1" = "true" ] && success_count=$((success_count + 1))
    [ "$success2" = "true" ] && success_count=$((success_count + 1))
    
    if [ "$success_count" = "1" ]; then
        pass_test "Exactly one concurrent update succeeded (race condition prevented)"
    else
        fail_test "Expected exactly 1 success, got $success_count (success1=$success1, success2=$success2)"
        return 1
    fi
    
    # Verify final state
    local get_data='{"key": "'$key'"}'
    response=$(make_request "$LEADER_URL/get" "$get_data")
    local final_version=$(get_version "$response")
    local final_value=$(get_value "$response")
    
    if [ "$final_version" = "2" ]; then
        pass_test "Final version is 2 after concurrent updates"
    else
        fail_test "Expected final version 2, got $final_version"
        return 1
    fi
    
    if [ "$final_value" = "$client1_value" ] || [ "$final_value" = "$client2_value" ]; then
        pass_test "Final value matches one of the concurrent updates: $final_value"
    else
        fail_test "Final value doesn't match any expected value: $final_value"
        return 1
    fi
    
    log_success "Concurrent update prevention working correctly"
}

# Test delete with versioning
test_versioned_delete() {
    log_test "Versioned Delete Operations"
    
    local key="delete_version_test_${TEST_ID}"
    local value="to_be_deleted"
    
    # Set initial value (new key creation with expected_version 0)
    local set_data='{"key": "'$key'", "value": "'$value'", "expected_version": 0, "id": "'$TEST_ID'"}'
    local response=$(make_request "$LEADER_URL/set" "$set_data")
    local version=$(get_version "$response")
    
    if [ "$version" = "1" ]; then
        pass_test "Initial value set for delete test"
    else
        fail_test "Failed to set initial value for delete test"
        return 1
    fi
    
    # Try to delete with wrong expected version (should fail)
    local wrong_delete='{"key": "'$key'", "expected_version": 2, "id": "'$TEST_ID'"}'
    response=$(make_request "$LEADER_URL/delete" "$wrong_delete" "false")
    local success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "false" ]; then
        pass_test "Delete with wrong expected version correctly rejected"
    else
        fail_test "Delete with wrong expected version should have failed"
        return 1
    fi
    
    # Verify key still exists
    local get_data='{"key": "'$key'"}'
    response=$(make_request "$LEADER_URL/get" "$get_data")
    local get_success=$(echo "$response" | jq -r '.success')
    
    if [ "$get_success" = "true" ]; then
        pass_test "Key still exists after failed delete"
    else
        fail_test "Key should still exist after failed delete"
        return 1
    fi
    
    # Delete with correct expected version (should succeed)
    local correct_delete='{"key": "'$key'", "expected_version": 1, "id": "'$TEST_ID'"}'
    response=$(make_request "$LEADER_URL/delete" "$correct_delete")
    success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "true" ]; then
        pass_test "Delete with correct expected version succeeded"
    else
        fail_test "Delete with correct expected version failed"
        return 1
    fi
    
    # Verify key is deleted
    response=$(make_request "$LEADER_URL/get" "$get_data" "false")
    get_success=$(echo "$response" | jq -r '.success')
    
    if [ "$get_success" = "false" ]; then
        pass_test "Key correctly deleted and no longer exists"
    else
        fail_test "Key should not exist after successful delete"
        return 1
    fi
    
    log_success "Versioned delete operations working correctly"
}

# Test metadata consistency
test_metadata_consistency() {
    log_test "Metadata Consistency and Timestamps"
    
    local key="metadata_test_${TEST_ID}"
    local value1="first"
    local value2="second"
    
    # Set initial value and capture timestamp (new key creation with expected_version 0)
    local set_data='{"key": "'$key'", "value": "'$value1'", "expected_version": 0, "id": "'$TEST_ID'"}'
    local response1=$(make_request "$LEADER_URL/set" "$set_data")
    local created1=$(echo "$response1" | jq -r '.metadata.created // ""')
    local updated1=$(echo "$response1" | jq -r '.metadata.updated // ""')
    local version1=$(get_version "$response1")
    
    if [ -n "$created1" ] && [ -n "$updated1" ] && [ "$version1" = "1" ]; then
        pass_test "Initial set has proper metadata (version=$version1)"
    else
        fail_test "Initial set metadata incomplete: created=$created1, updated=$updated1, version=$version1"
        return 1
    fi
    
    # Wait a moment to ensure timestamp difference
    sleep 1
    
    # Update value
    local update_data='{"key": "'$key'", "value": "'$value2'", "expected_version": 1, "id": "'$TEST_ID'"}'
    local response2=$(make_request "$LEADER_URL/set" "$update_data")
    local created2=$(echo "$response2" | jq -r '.metadata.created // ""')
    local updated2=$(echo "$response2" | jq -r '.metadata.updated // ""')
    local version2=$(get_version "$response2")
    
    if [ "$version2" = "2" ]; then
        pass_test "Update incremented version to 2"
    else
        fail_test "Update should increment version to 2, got $version2"
        return 1
    fi
    
    if [ "$created1" = "$created2" ]; then
        pass_test "Created timestamp remained consistent across updates"
    else
        fail_test "Created timestamp should remain consistent: $created1 != $created2"
        return 1
    fi
    
    if [ "$updated1" != "$updated2" ]; then
        pass_test "Updated timestamp changed on update"
    else
        fail_test "Updated timestamp should change on update"
        return 1
    fi
    
    # Verify GET also returns consistent metadata
    local get_data='{"key": "'$key'"}'
    local response3=$(make_request "$LEADER_URL/get" "$get_data")
    local created3=$(echo "$response3" | jq -r '.metadata.created // ""')
    local updated3=$(echo "$response3" | jq -r '.metadata.updated // ""')
    local version3=$(get_version "$response3")
    
    if [ "$created2" = "$created3" ] && [ "$updated2" = "$updated3" ] && [ "$version2" = "$version3" ]; then
        pass_test "GET returns consistent metadata with SET response"
    else
        fail_test "GET metadata inconsistent with SET response"
        return 1
    fi
    
    log_success "Metadata consistency verified"
}

# Test without expected version (new keys)
test_new_key_creation() {
    log_test "New Key Creation Without Expected Version"
    
    local key="new_key_test_${TEST_ID}"
    local value="new_value"
    
    # Create new key with expected_version 0
    local set_data='{"key": "'$key'", "value": "'$value'", "expected_version": 0, "id": "'$TEST_ID'"}'
    local response=$(make_request "$LEADER_URL/set" "$set_data")
    local version=$(get_version "$response")
    local success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "true" ] && [ "$version" = "1" ]; then
        pass_test "New key created without expected_version (version=$version)"
    else
        fail_test "New key creation failed: success=$success, version=$version"
        return 1
    fi
    
    # Try to update existing key without expected_version (should fail due to version conflict)
    local update_data='{"key": "'$key'", "value": "updated_without_version", "id": "'$TEST_ID'"}'
    response=$(make_request "$LEADER_URL/set" "$update_data" "false")
    success=$(echo "$response" | jq -r '.success')
    local error=$(echo "$response" | jq -r '.error // ""')
    
    if [ "$success" = "false" ] && [[ "$error" == *"version conflict"* ]]; then
        pass_test "Update without expected_version correctly rejected for existing key"
    else
        fail_test "Update without expected_version should fail for existing key: success=$success, error=$error"
        return 1
    fi
    
    log_success "New key creation behavior correct"
}

# Print test summary
print_test_summary() {
    echo
    echo "=================================================="
    echo "VERSIONING INTEGRATION TEST SUMMARY"
    echo "=================================================="
    echo -e "Tests Run:    ${BLUE}$TESTS_RUN${NC}"
    echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
    echo
    
    if [ "$TESTS_FAILED" -eq 0 ]; then
        echo -e "${GREEN}ALL TESTS PASSED! Versioning system is working correctly.${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed. Please review the output above.${NC}"
        return 1
    fi
}

# Main test execution
main() {
    echo "=================================================="
    echo "KV Store Versioning Integration Test"
    echo "Testing optimistic concurrency control and versioning"
    echo "=================================================="
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed. Please install jq."
        exit 1
    fi
    
    # Wait for services
    wait_for_services
    
    echo
    log_info "Starting versioning integration tests..."
    echo
    
    # Run all versioning tests
    test_basic_versioning
    echo
    
    test_version_conflicts  
    echo
    
    test_concurrent_updates
    echo
    
    test_versioned_delete
    echo
    
    test_metadata_consistency
    echo
    
    test_new_key_creation
    echo
    
    # Print summary and exit with appropriate code
    print_test_summary
}

# Execute main function
main "$@"

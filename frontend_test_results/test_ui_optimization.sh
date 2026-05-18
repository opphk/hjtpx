#!/bin/bash

# HJTPX Frontend UI Test Script
# This script tests the UI optimization changes by checking files directly

echo "========================================="
echo "HJTPX Frontend UI Optimization Test"
echo "========================================="
echo ""

# Test variables
FAILED=0
PASSED=0

# Function to test a file exists
test_file() {
    local file=$1
    local description=$2
    
    if [ -f "$file" ]; then
        echo "✓ PASSED: $description"
        echo "  File: $file"
        ((PASSED++))
        return 0
    else
        echo "✗ FAILED: $description"
        echo "  File not found: $file"
        ((FAILED++))
        return 1
    fi
}

# Function to check content in file
check_content() {
    local file=$1
    local pattern=$2
    local description=$3
    
    if [ ! -f "$file" ]; then
        echo "✗ FAILED: $description"
        echo "  File not found: $file"
        ((FAILED++))
        return 1
    fi
    
    if grep -q "$pattern" "$file"; then
        echo "✓ PASSED: $description"
        ((PASSED++))
        return 0
    else
        echo "✗ FAILED: $description"
        echo "  Pattern '$pattern' not found in $file"
        ((FAILED++))
        return 1
    fi
}

# Test CSS files
echo "1. Testing CSS Files:"
echo "-----------------------------------------"
test_file "/workspace/frontend/static/css/captcha-ui-unified.css" "Unified CSS file created"
test_file "/workspace/frontend/static/css/captcha-ui-optimized.css" "Original optimized CSS exists"
echo ""

# Test JS files
echo "2. Testing JavaScript Files:"
echo "-----------------------------------------"
test_file "/workspace/frontend/static/js/captcha-ui-unified.js" "Unified JS file created"
echo ""

# Test HTML templates use unified CSS
echo "3. Checking HTML Templates use Unified CSS:"
echo "-----------------------------------------"
check_content "/workspace/frontend/templates/home.html" "captcha-ui-unified.css" "home.html uses unified CSS"
check_content "/workspace/frontend/templates/captcha.html" "captcha-ui-unified.css" "captcha.html uses unified CSS"
check_content "/workspace/frontend/templates/3dcaptcha.html" "captcha-ui-unified.css" "3dcaptcha.html uses unified CSS"
check_content "/workspace/frontend/templates/lianliankan.html" "captcha-ui-unified.css" "lianliankan.html uses unified CSS"
check_content "/workspace/frontend/templates/seamless.html" "captcha-ui-unified.css" "seamless.html uses unified CSS"
check_content "/workspace/frontend/templates/voice-captcha.html" "captcha-ui-unified.css" "voice-captcha.html uses unified CSS"
echo ""

# Test unified JS is loaded
echo "4. Checking Unified JS is loaded in templates:"
echo "-----------------------------------------"
check_content "/workspace/frontend/templates/captcha.html" "captcha-ui-unified.js" "captcha.html loads unified JS"
check_content "/workspace/frontend/templates/seamless.html" "captcha-ui-unified.js" "seamless.html loads unified JS"
check_content "/workspace/frontend/templates/3dcaptcha.html" "captcha-ui-unified.js" "3dcaptcha.html loads unified JS"
check_content "/workspace/frontend/templates/lianliankan.html" "captcha-ui-unified.js" "lianliankan.html loads unified JS"
check_content "/workspace/frontend/templates/voice-captcha.html" "captcha-ui-unified.js" "voice-captcha.html loads unified JS"
echo ""

# Test unified JS contains required components
echo "5. Testing Unified JS Components:"
echo "-----------------------------------------"
check_content "/workspace/frontend/static/js/captcha-ui-unified.js" "ToastManager" "ToastManager component exists"
check_content "/workspace/frontend/static/js/captcha-ui-unified.js" "LoadingManager" "LoadingManager component exists"
check_content "/workspace/frontend/static/js/captcha-ui-unified.js" "AnimationManager" "AnimationManager component exists"
check_content "/workspace/frontend/static/js/captcha-ui-unified.js" "CaptchaUI" "CaptchaUI API exists"
echo ""

# Test unified CSS contains required styles
echo "6. Testing Unified CSS Features:"
echo "-----------------------------------------"
check_content "/workspace/frontend/static/css/captcha-ui-unified.css" "captcha-toast-container" "Toast container styles exist"
check_content "/workspace/frontend/static/css/captcha-ui-unified.css" "captcha-loading-spinner" "Loading animation styles exist"
check_content "/workspace/frontend/static/css/captcha-ui-unified.css" "captcha-toast" "Toast notification styles exist"
check_content "/workspace/frontend/static/css/captcha-ui-unified.css" "captcha-btn-primary" "Button styles exist"
check_content "/workspace/frontend/static/css/captcha-ui-unified.css" "captcha-result-banner" "Result banner styles exist"
check_content "/workspace/frontend/static/css/captcha-ui-unified.css" "@media" "Responsive styles exist"
check_content "/workspace/frontend/static/css/captcha-ui-unified.css" "prefers-reduced-motion" "Accessibility styles exist"
echo ""

# Test home.html integration
echo "7. Testing home.html Integration:"
echo "-----------------------------------------"
check_content "/workspace/frontend/templates/home.html" "CaptchaUI" "home.html integrates with CaptchaUI"
check_content "/workspace/frontend/templates/home.html" "showToast" "home.html uses showToast function"
echo ""

# Summary
echo "========================================="
echo "Test Summary"
echo "========================================="
echo "PASSED: $PASSED"
echo "FAILED: $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "✓ All tests passed!"
    echo ""
    echo "UI Optimization Summary:"
    echo "- Unified CSS file created with all required styles"
    echo "- Unified JS file created with Toast and Loading managers"
    echo "- All HTML templates updated to use unified CSS"
    echo "- Toast notification system implemented"
    echo "- Loading animations implemented"
    echo "- Responsive and accessible design ensured"
    exit 0
else
    echo "✗ Some tests failed. Please check the output above."
    exit 1
fi

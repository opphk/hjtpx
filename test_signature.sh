#!/bin/bash

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
SIGNATURE_KEY="your-signature-secret-key-change-in-production"
APP_ID="test_app"

generate_signature() {
    local method="$1"
    local path="$2"
    local body="$3"
    local timestamp="$4"
    local app_id="$5"

    local message="${method}:${path}:${body}:${timestamp}:${SIGNATURE_KEY}:${app_id}"
    echo -n "$message" | openssl dgst -sha256 -hmac "$SIGNATURE_KEY" | sed 's/^.* //'
}

generate_nonce() {
    head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32
}

echo "=== Testing Signature Verification ==="
echo ""

echo "1. Testing captcha/create endpoint with valid signature..."
timestamp=$(date +%s)
nonce=$(generate_nonce)
body="{\"app_id\":\"${APP_ID}\",\"captcha_type\":\"slider\",\"fingerprint\":\"test_fingerprint\"}"
signature=$(generate_signature "POST" "/api/v1/captcha/create" "$body" "$timestamp" "$APP_ID")

response=$(curl -s -X POST "${BASE_URL}/api/v1/captcha/create" \
    -H "Content-Type: application/json" \
    -H "X-Signature: ${signature}" \
    -H "X-Nonce: ${nonce}" \
    -H "X-Timestamp: ${timestamp}" \
    -H "X-App-ID: ${APP_ID}" \
    -H "X-Fingerprint: test_fingerprint" \
    -d "$body")

echo "Response: $response"
echo ""

if echo "$response" | grep -q '"code":0'; then
    echo "✓ Signature verification passed!"
else
    echo "✗ Signature verification failed!"
    exit 1
fi

echo ""
echo "2. Testing captcha/create endpoint without signature headers..."
response=$(curl -s -X POST "${BASE_URL}/api/v1/captcha/create" \
    -H "Content-Type: application/json" \
    -d '{"app_id":"test","captcha_type":"slider","fingerprint":"test"}')

echo "Response: $response"
echo ""

if echo "$response" | grep -q '"code":0'; then
    echo "⚠ Warning: Request without signature was accepted (signature may be disabled)"
else
    echo "✓ Request without signature was rejected"
fi

echo ""
echo "3. Testing captcha/create endpoint with invalid signature..."
timestamp=$(date +%s)
nonce=$(generate_nonce)
body='{"app_id":"test","captcha_type":"slider","fingerprint":"test"}'
signature="invalid_signature"

response=$(curl -s -X POST "${BASE_URL}/api/v1/captcha/create" \
    -H "Content-Type: application/json" \
    -H "X-Signature: ${signature}" \
    -H "X-Nonce: ${nonce}" \
    -H "X-Timestamp: ${timestamp}" \
    -H "X-App-ID: ${APP_ID}" \
    -d "$body")

echo "Response: $response"
echo ""

if echo "$response" | grep -q '"code":0'; then
    echo "✗ Invalid signature was accepted!"
    exit 1
else
    echo "✓ Invalid signature was rejected"
fi

echo ""
echo "=== All tests completed ==="

#!/bin/bash

BASE_URL="http://localhost:8080"

echo "=== Testing Behavior Validation System API ==="

echo ""
echo "1. Testing Health Check..."
curl -s "$BASE_URL/health" | jq .

echo ""
echo "2. Testing Captcha Creation..."
RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/captcha/create" \
  -H "Content-Type: application/json" \
  -d '{"type":"image","app_id":1}')
echo "$RESPONSE" | jq .
TOKEN=$(echo "$RESPONSE" | jq -r '.data.token')

echo ""
echo "3. Testing Captcha Status..."
curl -s "$BASE_URL/api/v1/captcha/status/$TOKEN" | jq .

echo ""
echo "4. Testing User Registration..."
curl -s -X POST "$BASE_URL/api/v1/user/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","username":"testuser"}' | jq .

echo ""
echo "5. Testing User Login..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/user/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}')
echo "$LOGIN_RESPONSE" | jq .
TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')

echo ""
echo "6. Testing User Profile (with auth)..."
curl -s "$BASE_URL/api/v1/user/profile" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""
echo "7. Testing Unauthorized Access..."
curl -s "$BASE_URL/api/v1/user/profile" | jq .

echo ""
echo "8. Testing Admin Login..."
curl -s -X POST "$BASE_URL/api/v1/admin/admin/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}' | jq .

echo ""
echo "=== API Tests Complete ==="

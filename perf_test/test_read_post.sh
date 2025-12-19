#!/bin/bash
set -e

REGISTER_URL="http://localhost:8080/api/auth/register"
TARGET_URL="http://localhost:8080"
COOKIE_FILE="cookies.txt"

rm -f "$COOKIE_FILE"

TIMESTAMP=$(date +%s)
EMAIL="testuser_${TIMESTAMP}@example.com"


echo "Email: $EMAIL"

curl -s -i -X POST "$REGISTER_URL" \
  -H "Content-Type: application/json" \
  -d "{
    \"firstName\": \"Alice\",
    \"lastName\": \"Smith\",
    \"email\": \"$EMAIL\",
    \"password\": \"password123\",
    \"confirmPassword\": \"password123\",
    \"dob\": \"1990-01-01T00:00:00Z\",
    \"gender\": \"female\"
  }" \
  -c "$COOKIE_FILE" > /dev/null



SESSION_ID=$(awk '$6=="session_id"{print $7}' "$COOKIE_FILE")
CSRF_TOKEN=$(awk '$6=="CSRF_token"{print $7}' "$COOKIE_FILE")



export SESSION_ID
export CSRF_TOKEN

echo "Session ID: $SESSION_ID"


wrk -t4 -c100 -d30s \
  -s ./perf_test/read_posts.lua \
  --timeout 10s \
  "$TARGET_URL"

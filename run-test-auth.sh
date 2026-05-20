#!/bin/bash

# --- Configuration ---
BASE_URL="http://localhost:8081/api/v1"
TIMESTAMP=$(date +%s)
REGISTER_EMAIL="tester_$TIMESTAMP@example.com"
REGISTER_PASSWORD="securepassword1"
UPDATE_EMAIL="updated_$TIMESTAMP@example.com"
NEW_PASS="newly_set_strong_pass_99"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== SteadyPhoto Auth Integration Test ===${NC}"

# 1. Register User
echo -e "\n[1/5] Registering user: $REGISTER_EMAIL..."
REG_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/auth/register" \
     -H "Content-Type: application/json" \
     -d "{\"email\": \"$REGISTER_EMAIL\", \"password\": \"$REGISTER_PASSWORD\"}")

REG_STATUS=$(echo "$REG_RESP" | tail -n1)
if [ "$REG_STATUS" == "201" ] || [ "$REG_STATUS" == "409" ]; then
    echo -e "${GREEN}SUCCESS${NC} (Status: $REG_STATUS)"
else
    echo -e "${RED}FAILED${NC} (Status: $REG_STATUS)"
    echo "$REG_RESP" | head -n -1
    exit 1
fi

# 2. Login
echo -e "\n[2/5] Logging in..."
LOGIN_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/auth/login" \
     -H "Content-Type: application/json" \
     -d "{\"email\": \"$REGISTER_EMAIL\", \"password\": \"$REGISTER_PASSWORD\"}")

LOGIN_STATUS=$(echo "$LOGIN_RESP" | tail -n1)
if [ "$LOGIN_STATUS" == "200" ]; then
    echo -e "${GREEN}SUCCESS${NC}"
    # Extract access_token using grep/sed (lightweight alternative to jq)
    ACCESS_TOKEN=$(echo "$LOGIN_RESP" | head -n -1 | grep -o '"access_token":"[^"]*' | sed 's/"access_token":"//')
    echo "Token acquired."
else
    echo -e "${RED}FAILED${NC} (Status: $LOGIN_STATUS)"
    echo "$LOGIN_RESP" | head -n -1
    exit 1
fi

# 3. Update Profile (Email)
echo -e "\n[3/5] Updating profile email to: $UPDATE_EMAIL..."
PATCH_EMAIL_RESP=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/auth/profile" \
     -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "Content-Type: application/json" \
     -d "{\"email\": \"$UPDATE_EMAIL\"}")

PATCH_EMAIL_STATUS=$(echo "$PATCH_EMAIL_RESP" | tail -n1)
if [ "$PATCH_EMAIL_STATUS" == "200" ]; then
    echo -e "${GREEN}SUCCESS${NC}"
else
    echo -e "${RED}FAILED${NC} (Status: $PATCH_EMAIL_STATUS)"
    echo "$PATCH_EMAIL_RESP" | head -n -1
    exit 1
fi

# 4. Update Profile (Password)
echo -e "\n[4/5] Updating password..."
PATCH_PASS_RESP=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/auth/profile" \
     -H "Authorization: Bearer $ACCESS_TOKEN" \
     -H "Content-Type: application/json" \
     -d "{\"password\": \"$NEW_PASS\"}")

PATCH_PASS_STATUS=$(echo "$PATCH_PASS_RESP" | tail -n1)
if [ "$PATCH_PASS_STATUS" == "200" ]; then
    echo -e "${GREEN}SUCCESS${NC}"
else
    echo -e "${RED}FAILED${NC} (Status: $PATCH_PASS_STATUS)"
    echo "$PATCH_PASS_RESP" | head -n -1
    exit 1
fi

# 5. List Media (Verification of Ownership/Auth)
echo -e "\n[5/5] Fetching media list (Ownership Check)..."
LIST_RESP=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/media" \
     -H "Authorization: Bearer $ACCESS_TOKEN")

LIST_STATUS=$(echo "$LIST_RESP" | tail -n1)
if [ "$LIST_STATUS" == "200" ]; then
    echo -e "${GREEN}SUCCESS${NC}"
    echo "Response Body:"
    echo "$LIST_RESP" | head -n -1 | python3 -m json.tool 2>/dev/null || echo "$LIST_RESP" | head -n -1
else
    echo -e "${RED}FAILED${NC} (Status: $LIST_STATUS)"
    echo "$LIST_RESP" | head -n -1
    exit 1
fi

# 6. Delete Account (Soft-Delete Test)
echo -e "\n[6/6] Deleting account..."
DELETE_RESP=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/auth/profile" \
     -H "Authorization: Bearer $ACCESS_TOKEN")

DELETE_STATUS=$(echo "$DELETE_RESP" | tail -n1)
if [ "$DELETE_STATUS" == "204" ] || [ "$DELETE_STATUS" == "200" ]; then
    echo -e "${GREEN}SUCCESS${NC}"
else
    echo -e "${RED}FAILED${NC} (Status: $DELETE_STATUS)"
    echo "$DELETE_RESP" | head -n -1
    exit 1
fi

# Verify account is disabled/unauthorized for subsequent requests
echo -e "\n[Verification] Attempting to use old token after deletion..."
VERIFY_RESP=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/media" \
     -H "Authorization: Bearer $ACCESS_TOKEN")

VERIFY_STATUS=$(echo "$VERIFY_RESP" | tail -n1)
if [ "$VERIFY_STATUS" == "401" ]; then
    echo -e "${GREEN}SUCCESS${NC} (Token correctly invalidated)"
else
    echo -e "${RED}FAILED${NC} (Status: $VERIFY_STATUS expected 401)"
    exit 1
fi

echo -e "\n${BLUE}=== All tests passed! ===${NC}"

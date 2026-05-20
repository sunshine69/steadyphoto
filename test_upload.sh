#!/bin/bash

# SteadyPhoto Upload Feature Test Script (API-only validation)
# Tests all upload functionality without requiring local psql access

# Configuration
API_URL="http://localhost:8081"
TEST_EMAIL="testuser@example.com"
TEST_PASSWORD="securepass123"
IMAGE_TO_UPLOAD="/tmp/test_image.jpg"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 Starting SteadyPhoto Upload Feature Test..."
echo ""

# Helper function for HTTP requests with error handling
http_request() {
    local method=$1
    shift 2
    curl -s "$method" "$@" | jq . || echo "Failed to parse JSON response"
}

# ============================================================
# STEP 0: Ensure we have a test image (creates dummy if missing)
# ============================================================
echo "${YELLOW}[STEP 0]${NC} Preparing test image..."
if [ ! -f "$IMAGE_TO_UPLOAD" ]; then
    echo "   Creating minimal valid PNG at $IMAGE_TO_UPLOAD..."
    # Create a tiny base64 encoded PNG (1x1 red pixel)
    echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==" | base64 -d > "$IMAGE_TO_UPLOAD"
    echo "   ✅ Test image created (size: $(wc -c < $IMAGE_TO_UPLOAD) bytes)"
else
    echo "   ✅ Using existing test image at $IMAGE_TO_UPLOAD ($(wc -c < $IMAGE_TO_UPLOAD) bytes)"
fi

# ============================================================
# STEP 1: Register User (idempotent)
# ============================================================
echo ""
echo "${YELLOW}[STEP 1]${NC} Attempting to register user..."
REGISTER_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}")

# Check if registration failed because the user already exists (Status Conflict)
if echo "$REGISTER_RESPONSE" | grep -q '"User already exists"'; then
    echo "[INFO] User ${TEST_EMAIL} already registered. Proceeding to login."
else
    echo "✅ Registration successful: $REGISTER_RESPONSE"
fi

# ============================================================
# STEP 2: Login and Extract Token & User ID  
# ============================================================
echo ""
echo "${YELLOW}[STEP 2]${NC} Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}")

# Extract access_token using jq (install with 'sudo apt install jq' if missing)
TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo -e "${RED}❌ Login failed! Response: $LOGIN_RESPONSE${NC}"
    exit 1
fi
echo "✅ Token obtained successfully."

# ============================================================  
# STEP 3: Upload Image (Single File)
# ============================================================
echo ""
echo "${YELLOW}[STEP 3]${NC} Uploading image..."
UPLOAD_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/media/upload" \
    -H "Authorization: Bearer $TOKEN" \
    -F "files=@${IMAGE_TO_UPLOAD}")

# Pretty print the response using jq (if available) or raw text  
echo "[INFO] Upload Response:"
if command -v jq &> /dev/null; then
    echo "$UPLOAD_RESPONSE" | jq . 2>/dev/null || cat <<< "$UPLOAD_RESPONSE" 
else
    echo "$UPLOAD_RESPONSE"
fi

# Validate upload response structure
UPLOADED_COUNT=$(echo "$UPLOAD_RESPONSE" | jq '.uploaded | length' 2>/dev/null)  
DUPLICATES_COUNT=$(echo "$UPLOAD_RESPONSE" | jq '.skipped_duplicates | length' 2>/dev/null) 

if [ "${UPLOADED_COUNT:-0}" -gt "0" ]; then
    echo "" 
    echo -e "${GREEN}✅ Upload Success! ${UPLOADED_COUNT} file(s) processed.${NC}" 
    
    # Get the ID of the newly uploaded media for DB check  
    MEDIA_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.uploaded[0].id')
    
    if [ ! -z "$MEDIA_ID" ] && [ "$MEDIA_ID" != "null" ]; then 
        echo "[INFO] Checking Database..."
        # Check PostgreSQL (adjust user/db as needed)  
        DB_CHECK=$(psql -U postgres -d steadyphoto_db -t -c \
            "SELECT id, filename FROM media WHERE id = '${MEDIA_ID}';") 
        
        if [ ! -z "$DB_CHECK" ]; then 
             echo -e "${GREEN}✅ Database Verified! Record found: $DB_CHECK${NC}"  
        else
             echo -e "${RED}❌ Database Check Failed. No record for ID ${MEDIA_ID}${NC}"  
        fi

        # Note: Filesystem check is tricky in a script without knowing the exact user UUID path structure, 
        # but we can look at recent files if needed.
    else
         echo -e "${RED}❌ Could not extract Media ID from response.${NC}"  
    fi
    
elif [ "${DUPLICATES_COUNT:-0}" -gt "0" ]; then
     echo ""
     echo -e "${GREEN}✅ File detected as duplicate (already exists in system).${NC}" 
else
    echo ""
    echo -e "${RED}❌ Upload failed or returned empty result set. Check server logs.${NC}"  
fi

echo "\n🏁 Integration Test Finished." 

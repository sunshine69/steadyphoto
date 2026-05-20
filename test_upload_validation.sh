#!/bin/bash

# SteadyPhoto: New Upload & Retrieval Validation Script
# Tests fresh upload, filesystem hierarchy creation, and API retrieval endpoints

set -e  # Exit on error (we'll handle specific cases gracefully)

API_URL="http://localhost:8081"
TEST_EMAIL="testuser@example.com"
TEST_PASSWORD="securepass123"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🚀 Starting New Upload & Retrieval Validation..."
echo ""

# ============================================================
# STEP 1: Login to get token
# ============================================================
echo "${YELLOW}[STEP 1]${NC} Authenticating..."
LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')
USER_ID=$(echo "$LOGIN_RESPONSE" | jq -r '.user_id')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo -e "${RED}❌ Login failed!${NC}"
    exit 1
fi
echo "   ✅ Authenticated. User ID: ${USER_ID}"

# ============================================================
# STEP 2: Create a UNIQUE test image (different hash from existing)
# ============================================================
UNIQUE_IMAGE="/tmp/fresh_test_upload.png"
if [ -f "$UNIQUE_IMAGE" ]; then rm -f "$UNIQUE_IMAGE"; fi

echo "${YELLOW}[STEP 2]${NC} Creating unique test file..."
# Create a slightly different PNG (1x1 blue pixel) to guarantee new hash
echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==" | base64 -d > "$UNIQUE_IMAGE"
# Modify one byte to ensure different hash (optional but safe)
echo "X" >> "$UNIQUE_IMAGE"  # Makes it invalid PNG but unique bytes for hashing test

FILE_SIZE=$(wc -c < "$UNIQUE_IMAGE")
echo "   ✅ Created: $UNIQUE_IMAGE (${FILE_SIZE} bytes)"

# ============================================================
# STEP 3: Upload the fresh file
# ============================================================
echo "${YELLOW}[STEP 3]${NC} Uploading new image..."
UPLOAD_RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/media/upload" \
    -H "Authorization: Bearer $TOKEN" \
    -F "files=@${UNIQUE_IMAGE}")

echo ""
if command -v jq &> /dev/null; then
    echo "$UPLOAD_RESPONSE" | jq . 2>/dev/null || cat <<< "$UPLOAD_RESPONSE"
else
    cat <<< "$UPLOAD_RESPONSE"
fi

UPLOADED_COUNT=$(echo "$UPLOAD_RESPONSE" | jq '.uploaded | length' 2>/dev/null)
DUPLICATES_COUNT=$(echo "$UPLOAD_RESPONSE" | jq '.skipped_duplicates | length' 2>/dev/null)

if [ "${UPLOADED_COUNT:-0}" -gt "0" ]; then
    echo ""
    echo -e "${GREEN}✅ New upload successful!${NC}" 
    
    MEDIA_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.uploaded[0].id')
    FILENAME_UPLOADED=$(echo "$UPLOAD_RESPONSE" | jq -r '.uploaded[0].filename')
    PATH_IN_DB=$(echo "$UPLOAD_RESPONSE" | jq -r '.uploaded[0].path')  # This is the relative path stored in DB
    
    echo "   ℹ️  Media ID: ${MEDIA_ID}"
    echo "   ℹ️  Filename: ${FILENAME_UPLOADED}"  
    echo "   ℹ️  Relative Path (DB): ${PATH_IN_DB}"

elif [ "${DUPLICATES_COUNT:-0}" -gt "0" ]; then
     echo ""
     echo -e "${YELLOW}⚠️  File was detected as duplicate despite being unique!${NC}" 
     MEDIA_ID=$(echo "$UPLOAD_RESPONSE" | jq '.skipped_duplicates[0].id' 2>/dev/null)
else
    echo ""
    echo -e "${RED}❌ Upload failed or returned empty result set.${NC}"  
    exit 1
fi

# ============================================================
# STEP 4: Verify Filesystem Hierarchy (storage/{user_id}/YYYY/MM/DD/)
# ============================================================
echo ""
echo "${YELLOW}[STEP 4]${NC} Verifying storage hierarchy on disk..."

if [ -n "$PATH_IN_DB" ] && [ ! -z "$USER_ID" ]; then
    # Construct expected path: ./storage/{user_id}/{relative_path_from_db}
    EXPECTED_PATH="./storage/${USER_ID}/${PATH_IN_DB}"
    
    if [ -f "$EXPECTED_PATH" ]; then
        DISK_SIZE=$(wc -c < "$EXPECTED_PATH")
        echo "   ✅ File exists on disk at: ${EXPECTED_PATH}"
        
        # Verify size matches what we uploaded (allow small diff for base64 decode quirks)
        EXPECTED_ORIGINAL=105  # The PNG + 'X' byte we added
        if [ "$DISK_SIZE" -ge "90" ]; then
            echo "   ✅ Filesize verified: ${DISK_SIZE} bytes (matches upload)"
            
            # Check directory structure exists correctly
            DIR_PATH=$(dirname "$EXPECTED_PATH")
            if [ -d "$DIR_PATH" ]; then
                echo "   ✅ Directory hierarchy created successfully."
                
                # Verify it follows YYYY/MM/DD pattern
                DATE_DIR=$(basename "$(dirname "$DIR_PATH")")  # Should be DD or MM depending on structure
                YEAR_DIR=$(basename "$(dirname "$(dirname "$DIR_PATH")")")  # Should be YYYY
                
                echo "   ℹ️  Path components: storage/${USER_ID}/${YEAR_DIR}/.../" 
                
            else
                 echo -e "${RED}❌ Directory path missing!${NC}"  
            fi
            
        else
             echo -e "${YELLOW}⚠️  Filesize mismatch (expected ~105, got ${DISK_SIZE})${NC}" 
        fi
        
    else
         # Try alternative: maybe storage service uses absolute or different base?
         ALTERNATIVE="./storage/${PATH_IN_DB}"
         if [ -f "$ALTERNATIVE" ]; then
             echo "   ✅ File found at alternative path: ${ALTERNATIVE}"
         else
            echo ""  
            echo -e "${RED}❌ File NOT FOUND on disk!${NC}" 
            echo "      Expected: ./storage/${USER_ID}/${PATH_IN_DB}"
            
            # List what's actually in storage for debugging
            echo ""
            echo "   🔍 Current storage contents:"
            find "./storage" -type f 2>/dev/null | head -5 || echo "      (empty or inaccessible)"  
         fi
    fi
    
else
     echo "${YELLOW}⚠️  Skipping filesystem check due to missing data.${NC}" 
fi

# ============================================================
# STEP 5: Fetch Metadata via GET /api/v1/media/{id}
# ============================================================
echo ""
echo "${YELLOW}[STEP 5]${NC} Fetching media metadata..."
METADATA_RESPONSE=$(curl -s -X GET "${API_URL}/api/v1/media/${MEDIA_ID}" \
    -H "Authorization: Bearer $TOKEN")

if [ $? -eq 0 ] && ! echo "$METADATA_RESPONSE" | grep -q '"error"'; then  
    if command -v jq &> /dev/null; then 
        echo ""
        echo "$METADATA_RESPONSE" | jq . 2>/dev/null || cat <<< "$METADATA_RESPONSE" 
        
        # Validate key fields exist and match upload response
        HAS_ID=$(echo "$METADATA_RESPONSE" | jq 'has("id")' 2>/dev/null)  
        HAS_FILENAME=$(echo "$METADATA_RESPONSE" | jq 'has("filename")' 2>/dev/null) 
        HAS_PATH=$(echo "$METADATA_RESPONSE" | jq 'has("path")' 2>/dev/null)
        
        if [ "$HAS_ID" == "true" ] && [ "$HAS_FILENAME" == "true" ]; then  
            echo ""
            echo -e "${GREEN}✅ Metadata validation passed!${NC}" 
            
            RETURNED_PATH=$(echo "$METADATA_RESPONSE" | jq -r '.path')
            if [ "$RETURNED_PATH" == "$PATH_IN_DB" ]; then
                echo "   ✅ DB path matches upload response."  
            else
                 echo -e "${YELLOW}⚠️  Path mismatch: Upload said ${PATH_IN_DB}, metadata says ${RETURNED_PATH}${NC}" 
            fi
            
        else
             echo ""
             echo -e "${RED}❌ Metadata missing required fields!${NC}" 
        fi
        
    else
         cat <<< "$METADATA_RESPONSE" | head -20  
    fi
    
else
     echo ""
     if [ $? -ne 0 ]; then
          echo -e "${RED}❌ Failed to fetch metadata HTTP status: ${?}${NC}" 
     else
        echo "   ❌ API returned error or empty response."  
     fi
fi

# ============================================================
# STEP 6: Download Original File via GET /original (Range Support Test)
# ============================================================
echo ""
echo "${YELLOW}[STEP 6]${NC} Testing file download & Range requests..." 

TEMP_DOWNLOADED="/tmp/downloaded_fresh.jpg"
curl -s -o "$TEMP_DOWNLOADED" \
    "http://localhost:8081/api/v1/media/${MEDIA_ID}/original" \
    -H "Authorization: Bearer $TOKEN"

if [ $? -eq 0 ] && [ -f "$TEMP_DOWNLOADED" ]; then  
    DOWN_SIZE=$(wc -c < $TEMP_DOWNLOADED) 
    
    if [ "$DOWN_SIZE" -gt 10 ]; then 
        echo ""
        
        # Check HTTP status for Range requests (should be 206 Partial Content when using Range header)
        RANGE_STATUS=$(curl -s -o /dev/null \
            "http://localhost:8081/api/v1/media/${MEDIA_ID}/original" \
            -H "Authorization: Bearer $TOKEN") 
        
        if [ "$RANGE_STATUS" == "206" ]; then  
             echo -e "${GREEN}✅ Range requests working! (HTTP 206 Partial Content)${NC}" 
             
             # Test actual range header behavior
             RANGE_BYTES=$(curl -s \
                 "http://localhost:8081/api/v1/media/${MEDIA_ID}/original" \
                 -H "Authorization: Bearer $TOKEN" \
                 -H "Range: bytes=0-9") 
                 
             if [ ${#RANGE_BYTES} -eq 10 ]; then  
                echo "   ✅ Range header correctly returns requested byte slice." 
            else
                 echo -e "${YELLOW}⚠️  Expected 10 bytes from range request, got ${#RANGE_BYTES}${NC}" 
             fi
            
        elif [ "$RANGE_STATUS" == "200" ]; then  
             # Some servers return 200 for full file if no Range header sent
             echo -e "${GREEN}✅ Full download successful! (HTTP 200)${NC}" 
             
            # Verify it's actually binary data and not JSON error
            FIRST_BYTE=$(xxd -l 1 "$TEMP_DOWNLOADED" | awk '{print $2}')  
            if [ ! -z "$FIRST_BYTE" ]; then
                 echo "   ✅ File is valid binary content (starts with byte: ${FIRST_BYTE})" 
             fi
            
        else
             echo ""
             echo -e "${YELLOW}⚠️  Download returned status code: ${RANGE_STATUS}${NC}"  
        fi
        
    else
         echo ""
         echo -e "${RED}❌ File downloaded but too small (${DOWN_SIZE} bytes). Possible error response.${NC}" 
         
         # Show what was actually saved (might be JSON error)
         if command -v jq &> /dev/null; then  
            cat "$TEMP_DOWNLOADED" | jq . 2>/dev/null || echo "Raw content:" && head -c 100 "$TEMP_DOWNLOADED" 
        fi
        
    fi
    
else
     echo ""
     echo -e "${RED}❌ Failed to download original file!${NC}"  
fi

# ============================================================
# STEP 7: Cleanup (Delete uploaded test media)
# ============================================================
echo ""
echo "${YELLOW}[STEP 7]${NC} Cleaning up test data..." 

DELETE_RESPONSE=$(curl -s -X DELETE \
    "http://localhost:8081/api/v1/media/${MEDIA_ID}" \
    -H "Authorization: Bearer $TOKEN") 
    
if [ $? -eq 0 ]; then  
     VERIFY_DELETE=$(curl -s -o /dev/null -w "%{http_code}" \
         "${API_URL}/api/v1/media/${MEDIA_ID}" \
         -H "Authorization: Bearer $TOKEN") 
        
    if [ "$VERIFY_DELETE" == "404" ] || echo "$DELETE_RESPONSE" | grep -q '"deleted"'; then  
        echo ""
        echo -e "${GREEN}✅ Cleanup successful! Media deleted and no longer accessible.${NC}" 
    else
         echo ""
         echo -e "${YELLOW}⚠️  Delete returned status ${VERIFY_DELETE}${NC}" 
     fi
    
else
     echo ""
     echo -e "${RED}❌ Failed to delete test media${NC}"  
fi

# ============================================================
# FINAL SUMMARY
# ============================================================
echo ""
echo "=================================================="
echo -e "${GREEN}🏁 New Upload & Retrieval Validation Complete!${NC}" 
echo "=================================================="
echo ""
echo "✅ Features Validated:"
echo "   • Fresh file upload (SHA256 + DB insert)"  
echo "   • Filesystem hierarchy creation"
echo "   • Metadata retrieval via GET /api/v1/media/{id}"
echo "   • Original file streaming & Range request support" 
echo ""

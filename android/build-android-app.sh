#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "=============================================="
echo "   SteadyPhoto Android Build Script"
echo "=============================================="

# Get the directory where this script is located (the android folder)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 1. Build Gomobile Bindings
echo ""
echo "--- Step 1: Building Go Mobile Bindings ---"
echo "Target: gomobile-bindings -> android/app/libs/mobile-bindings.aar"

if command -v gomobile &> /dev/null; then
    # Move into the gomobile-bindings directory (where go.mod lives)
    cd "$PROJECT_ROOT/android/gomobile-bindings"

    echo "🔧 Setting 16KB ELF alignment flags for Gomobile..."

    # Include default compiler flags (-O2) alongside the 16KB alignment parameters
    # -Wl,-z,max-page-size=16384 is the standard linker flag for 16KB alignment
    export CGO_CFLAGS="-O2"
    export CGO_LDFLAGS="-O2 -s -w -Wl,-z,max-page-size=16384"

    # Explicitly specify -androidapi to force NDK toolchain alignment compliance
    # Also passing it via -ldflags for the Go linker
    gomobile bind \
        -v \
        -target android/arm64,android/amd64 \
        -androidapi 21 \
        -ldflags="-extldflags=-Wl,-z,max-page-size=16384 -s -w" \
        -o "$SCRIPT_DIR/app/libs/mobile-bindings.aar" ./mobile

    echo "✅ Gomobile bindings built successfully."
else
    echo "⚠️ Warning: 'gomobile' command not found in your PATH."
fi

# 2. Build Android APK via Gradle
echo ""
echo "--- Step 2: Cleaning and Building Debug APK ---"
cd "$SCRIPT_DIR"

# Clean previous builds to ensure a fresh state
./gradlew clean

# Run the assembly task
./gradlew assembleDebug

if [ $? -eq 0 ]; then
    APK_PATH="$SCRIPT_DIR/app/build/outputs/apk/debug/app-debug.apk"
    echo ""
    echo "=============================================="
    echo "✅ BUILD SUCCESSFUL!"
    echo "----------------------------------------------"
    echo "📍 APK Location: $APK_PATH"
    echo ""
    echo "🚀 Quick Install Command:"
    echo "   adb install \"$APK_PATH\""
    echo "=============================================="
else
    echo ""
    echo "❌ BUILD FAILED!"
    exit 1
fi

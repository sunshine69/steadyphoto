#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "=============================================="
echo "   SteadyPhoto Android Build Script"
echo "=============================================="

# Usage information
usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --debug     Build a debug APK (default)"
    echo "  --release   Build a production-ready release APK"
    echo "  --arm64     Target arm64-v8a only (Android phones/tablets)"
    echo "  --x86_64    Target x86_64 only (Intel/AMD Android emulators)"
    echo "  --help      Display this help message"
}

# Parse arguments
BUILD_TYPE="debug"
ABI_FILTER=""

#arm64-v8a (Mandatory standard for all modern 64-bit mobile devices)
#armeabi-v7a (Legacy 32-bit mobile fallback)
#x86_64 (64-bit desktop/emulator target)
#x86 (Legacy 32-bit desktop/emulator target)

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --debug) BUILD_TYPE="debug"; shift ;;
        --release) BUILD_TYPE="release"; shift ;;
        # add armeabi-v7a if want to support fall back for OxygenOS happy?
        --arm64)
        ABI_FILTER="arm64-v8a"
        GOMOBILE_TARGET="android/arm64"
        shift
        ;;
        --x86_64)
            ABI_FILTER="x86_64"
            GOMOBILE_TARGET="android/amd64" # The 32 bit version is android/386
        shift
        ;;
        --help) usage; exit 0 ;;
        *) echo "Unknown parameter passed: $1"; usage; exit 1 ;;
    esac
done

# Set default ABI filter if not specified (arm64 for mobile, both for desktop)
if [ -z "$ABI_FILTER" ]; then
    # For mobile-focused builds, default to arm64
    # If you want both ABIs, call with: --debug or --release without --arm64/--x86_64
    ABI_FILTER="arm64-v8a,x86_64"
    GOMOBILE_TARGET="android/arm64,android/amd64"
fi

# Get the directory where this script is located (the android folder)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Target Build Type: ${BUILD_TYPE^^}"
echo "Target Platform(s): $ABI_FILTER"

# 1. Build Gomobile Bindings (Same for both debug and release)
echo ""
echo "--- Step 1: Building Go Mobile Bindings ---"
echo "Target: gomobile-bindings -> android/app/libs/mobile-bindings.aar"

if command -v gomobile &> /dev/null; then
    # Move into the gomobile-bindings directory (where go.mod lives)
    cd "$PROJECT_ROOT/android/gomobile-bindings"

    echo "🔧 Setting 16KB ELF alignment flags for Gomobile..."

    # Include default compiler flags (-O2) alongside the 16KB alignment parameters
    export CGO_CFLAGS="-O2"
    export CGO_LDFLAGS="-O2 -s -w -Wl,-z,max-page-size=16384"
# To support 32 bit arm need to add 'android/arm' but then max-page-size=16384 above wont work as 32 bit use 4k default. Thus probably drop Oxygen Support for all
    gomobile bind \
        -v \
        -target $GOMOBILE_TARGET \
        -androidapi 21 \
        -ldflags="-extldflags=-Wl,-z,max-page-size=16384 -s -w" \
        -o "$SCRIPT_DIR/app/libs/mobile-bindings.aar" ./mobile

    echo "✅ Gomobile bindings built successfully."
else
    echo "⚠️ Warning: 'gomobile' command not found in your PATH."
fi

# 2. Build Android APK via Gradle
echo ""
echo "--- Step 2: Cleaning and Building ${BUILD_TYPE^^} APK ---"
cd "$SCRIPT_DIR"

# Clean previous builds to ensure a fresh state
./gradlew clean

# Run the assembly task (assembleDebug or assembleRelease)
GRADLE_TASK="assemble${BUILD_TYPE^}"
echo "Running: ./gradlew $GRADLE_TASK -PabiFilter=$ABI_FILTER"
./gradlew "$GRADLE_TASK" -PabiFilter="$ABI_FILTER"

if [ $? -eq 0 ]; then
    # Determine path based on build type — APKs are now named steadyphoto-<variant>.apk
    APK_PATH="$SCRIPT_DIR/app/build/outputs/apk/$BUILD_TYPE/steadyphoto-$BUILD_TYPE.apk"

    # Fallback: find any apk in the output directory
    if [ ! -f "$APK_PATH" ]; then
        APK_PATH=$(find "$SCRIPT_DIR/app/build/outputs/apk" -name "steadyphoto-*.apk" | head -n 1)
    fi

    echo ""
    echo "=============================================="
    echo "✅ BUILD SUCCESSFUL!"
    echo "----------------------------------------------"
    echo "📍 APK Location: $APK_PATH"
    echo ""
    if [ "$BUILD_TYPE" == "release" ]; then
        echo "🚀 Note: This is a RELEASE build (minified/obfuscated) and signed with the release keystore."
    else
        echo "🚀 Quick Install Command:"
        echo "   adb install \"$APK_PATH\""
    fi
    echo "=============================================="
else
    echo ""
    echo "❌ BUILD FAILED!"
    exit 1
fi


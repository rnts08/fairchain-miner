#!/bin/bash

set -euo pipefail

# Check if gh CLI is installed
if ! command -v gh &> /dev/null
then
    echo "GitHub CLI (gh) is not installed. Please install it to continue."
    echo "See: https://cli.github.com/"
    exit 1
fi

# Get current version from main.go
CURRENT_VERSION=$(grep -oP 'const version = "\K[^"]+' cmd/fairchain-miner/main.go)
echo "Current miner version: v${CURRENT_VERSION}"

# Prompt for new version
read -p "Enter the new version (e.g., 1.0.0): v" NEW_VERSION

if [[ -z "$NEW_VERSION" ]]; then
    echo "New version cannot be empty. Aborting."
    exit 1
fi

# Validate version format (simple check)
if ! [[ "$NEW_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Invalid version format. Please use X.Y.Z (e.g., 1.0.0). Aborting."
    exit 1
fi

FULL_TAG="v${NEW_VERSION}"

echo "Preparing to create and push tag: ${FULL_TAG}"

# Create git tag and push to trigger GitHub Actions release workflow
git tag -a "${FULL_TAG}" -m "Release ${FULL_TAG}"
git push origin "${FULL_TAG}"

echo "Release process initiated for ${FULL_TAG}."
echo "Monitor the GitHub Actions workflow for 'Release' to track progress."
echo "Once the workflow completes, check the GitHub Releases page."
#!/bin/bash
# Wait for GitHub Actions build to complete

echo "Waiting for build to complete..."
while true; do
    STATUS=$(gh run list -w "Build and Release" -L 1 --json status,conclusion --jq '.[0]')
    RUN_STATUS=$(echo "$STATUS" | jq -r '.status')
    CONCLUSION=$(echo "$STATUS" | jq -r '.conclusion')
    
    if [ "$RUN_STATUS" = "completed" ]; then
        echo "Build completed with conclusion: $CONCLUSION"
        if [ "$CONCLUSION" = "success" ]; then
            echo "✅ Build successful!"
            exit 0
        else
            echo "❌ Build failed!"
            gh run view --log-failed
            exit 1
        fi
    else
        echo "Build still running... (status: $RUN_STATUS)"
        sleep 30
    fi
done

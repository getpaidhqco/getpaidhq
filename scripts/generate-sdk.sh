#!/bin/bash

set -e

echo "🚀 Generating GetPaidHQ TypeScript SDK..."

# Ensure we're in the right directory
cd "$(dirname "$0")/.."

# Check if swagger.yml exists
if [ ! -f "swagger.yml" ]; then
    echo "❌ swagger.yml not found in root directory"
    exit 1
fi

# Generate the SDK
cd sdks/typescript
pnpm install
pnpm run generate
pnpm run build

echo "✅ SDK generation completed!"
echo "📦 Ready for publishing: sdks/typescript/dist"
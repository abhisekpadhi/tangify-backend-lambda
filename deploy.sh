#!/bin/bash

set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"

echo "Building..."
sam build

echo "Deploying..."
sam deploy --no-confirm-changeset

echo "Syncing Lambda environment variables..."
"$ROOT/scripts/sync-lambda-env.sh"

echo "Deployment complete"
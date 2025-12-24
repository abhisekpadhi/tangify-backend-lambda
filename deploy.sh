#!/bin/bash

set -e

echo "Building..."
sam build

echo "Deploying..."
sam deploy --no-confirm-changeset

echo "Deployment complete"
#!/bin/bash

# Temporal Schema Upgrade Script
# Upgrades Temporal schema from 1.27.0 to 1.28.0

set -e

echo "=== Temporal Schema Upgrade Script ==="
echo "This script will upgrade your Temporal schema from 1.27.0 to 1.28.0"
echo ""

# Configuration
NETWORK="payloop-network"
DB_HOST="postgresql"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="postgres"
TEMPORAL_ADDRESS="temporal:7233"

echo "Please ensure:"
echo "1. Your Temporal services are running (docker-compose up -d)"
echo "2. You have a backup of your database"
echo ""
read -p "Do you want to continue? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]
then
    echo "Upgrade cancelled."
    exit 1
fi

echo ""
echo "Starting schema upgrade..."

# Run schema update using the current admin tools version
docker run --rm \
  --network "$NETWORK" \
  -e TEMPORAL_CLI_ADDRESS="$TEMPORAL_ADDRESS" \
  temporalio/admin-tools:1.27.0 \
  temporal operator schema update \
  --schema-dir /etc/temporal/schema/postgresql/v12/temporal \
  --db postgres \
  --db-host "$DB_HOST" \
  --db-port "$DB_PORT" \
  --db-user "$DB_USER" \
  --db-password "$DB_PASSWORD" \
  --version 1.28

echo ""
echo "Updating visibility schema..."

docker run --rm \
  --network "$NETWORK" \
  -e TEMPORAL_CLI_ADDRESS="$TEMPORAL_ADDRESS" \
  temporalio/admin-tools:1.27.0 \
  temporal operator schema update \
  --schema-dir /etc/temporal/schema/postgresql/v12/visibility \
  --db postgres \
  --db-host "$DB_HOST" \
  --db-port "$DB_PORT" \
  --db-user "$DB_USER" \
  --db-password "$DB_PASSWORD" \
  --version 1.28

echo ""
echo "Schema upgrade completed successfully!"
echo ""
echo "Next steps:"
echo "1. Stop Temporal services: docker-compose -f docker/docker-compose.yml down"
echo "2. Start with new versions: docker-compose -f docker/docker-compose.yml up -d"
echo ""
echo "The new versions in docker/.env have already been updated to:"
echo "- TEMPORAL_VERSION=1.28.0"
echo "- TEMPORAL_ADMINTOOLS_VERSION=1.28.0"
echo "- TEMPORAL_UI_VERSION=2.36.0"
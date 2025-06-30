#!/bin/bash

# Script to run integration tests with proper setup

echo "=== Payloop Integration Test Runner ==="
echo

# Check if docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

echo "✓ Docker is running"

# Start required services
echo "Starting required services..."
docker-compose up -d postgres redis nats

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 5

# Check if services are healthy
echo "Checking service health..."
docker-compose ps

# Run database migrations
echo "Running database migrations..."
pnpm dlx prisma generate
pnpm dlx prisma db push

# Create test namespace for Temporal
echo "Creating Temporal namespace..."
docker-compose exec -T temporal temporal operator namespace create -n subscriptions 2>/dev/null || echo "Namespace already exists"

# Run the integration tests
echo
echo "=== Running Integration Tests ==="
echo

# Set test environment variables
export GETPAIDHQ_ENV=test
export GETPAIDHQ_DATABASE_URL="postgresql://payloop:payloop@localhost:5432/payloop_test?schema=public"
export GETPAIDHQ_REPORTING_DATABASE_URL="postgresql://payloop:payloop@localhost:5433/payloop_reporting_test?schema=public"

# Run specific integration test
go test -v ./internal/testing/integration -run TestRealSubscriptionFlow

echo
echo "=== Test Complete ==="
echo

# Optional: Stop services after test
read -p "Stop services? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]
then
    docker-compose down
    echo "✓ Services stopped"
fi
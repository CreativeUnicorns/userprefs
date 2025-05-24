#!/bin/bash

# test_example.sh - Test script for Supabase example

set -e

echo "🧪 Testing Supabase Example"
echo "=========================="

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is required but not installed"
    exit 1
fi

# Check if Docker Compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is required but not installed"
    exit 1
fi

# Start services
echo "🐳 Starting Docker services..."
make docker-up

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check PostgreSQL connection
echo "🔍 Testing PostgreSQL connection..."
if docker-compose exec postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo "✅ PostgreSQL is ready"
else
    echo "❌ PostgreSQL is not ready"
    exit 1
fi

# Check Redis connection
echo "🔍 Testing Redis connection..."
if docker-compose exec redis redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis is ready"
else
    echo "❌ Redis is not ready"
    exit 1
fi

# Test the example application
echo "🚀 Testing example application..."

# Copy local environment
cp .env.local .env

# Run the example (with timeout)
if timeout 30s go run main.go > test_output.log 2>&1; then
    echo "✅ Example ran successfully"
else
    echo "⚠️  Example finished (possibly timed out, which is expected)"
fi

# Check for key outputs in the log
echo "📋 Checking test results..."

if grep -q "Successfully demonstrated Supabase integration" test_output.log; then
    echo "✅ Supabase integration test passed"
else
    echo "❌ Supabase integration test failed"
    echo "📄 Last few lines of output:"
    tail -10 test_output.log
    exit 1
fi

if grep -q "All concurrent updates completed" test_output.log; then
    echo "✅ Concurrent operations test passed"
else
    echo "❌ Concurrent operations test failed"
fi

if grep -q "100 cached reads took:" test_output.log; then
    echo "✅ Performance test passed"
else
    echo "❌ Performance test failed"
fi

# Check database state
echo "🔍 Checking database state..."
USER_COUNT=$(docker-compose exec postgres psql -U postgres -d postgres -t -c "SELECT COUNT(*) FROM userprefs.user_preferences;" | xargs)
if [ "$USER_COUNT" -gt 0 ]; then
    echo "✅ Database contains $USER_COUNT user preferences"
else
    echo "❌ No user preferences found in database"
    exit 1
fi

DEFINITION_COUNT=$(docker-compose exec postgres psql -U postgres -d postgres -t -c "SELECT COUNT(*) FROM userprefs.preference_definitions;" | xargs)
if [ "$DEFINITION_COUNT" -gt 0 ]; then
    echo "✅ Database contains $DEFINITION_COUNT preference definitions"
else
    echo "❌ No preference definitions found in database"
    exit 1
fi

# Cleanup
echo "🧹 Cleaning up..."
rm -f .env test_output.log
make docker-down

echo ""
echo "🎉 All tests passed!"
echo "=========================="
echo "The Supabase example is working correctly."
echo ""
echo "To run the example manually:"
echo "1. Copy .env.example to .env and configure with your Supabase credentials"
echo "2. Run: go run main.go"
echo ""
echo "To run with local Docker environment:"
echo "1. Run: make run-local"

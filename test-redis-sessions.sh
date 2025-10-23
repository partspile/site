#!/bin/bash

# Test Redis Sessions
# This script tests that Redis sessions are working correctly

echo "Testing Redis Sessions..."

# Check if Redis is accessible
echo "1. Testing Redis connection..."
if ! redis-cli -h "$REDIS_ADDRESS" -a "$REDIS_PASSWORD" ping > /dev/null 2>&1; then
    echo "❌ Redis connection failed. Make sure REDIS_ADDRESS and REDIS_PASSWORD are set correctly."
    exit 1
fi
echo "✅ Redis connection successful"

# Check if environment variables are set
echo "2. Checking environment variables..."
if [ -z "$REDIS_ADDRESS" ]; then
    echo "❌ REDIS_ADDRESS environment variable is not set"
    exit 1
fi

if [ -z "$REDIS_PASSWORD" ]; then
    echo "⚠️  REDIS_PASSWORD environment variable is not set (this might be intentional)"
fi

echo "✅ Environment variables configured"

# Test Redis key operations
echo "3. Testing Redis key operations..."
redis-cli -h "$REDIS_ADDRESS" -a "$REDIS_PASSWORD" set "test:session" "test_value" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Redis SET operation successful"
else
    echo "❌ Redis SET operation failed"
    exit 1
fi

redis-cli -h "$REDIS_ADDRESS" -a "$REDIS_PASSWORD" get "test:session" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Redis GET operation successful"
else
    echo "❌ Redis GET operation failed"
    exit 1
fi

# Clean up test key
redis-cli -h "$REDIS_ADDRESS" -a "$REDIS_PASSWORD" del "test:session" > /dev/null 2>&1

echo ""
echo "🎉 Redis sessions setup is ready!"
echo ""
echo "To test the application:"
echo "1. Set your environment variables:"
echo "   export REDIS_ADDRESS=your-redis-host:port"
echo "   export REDIS_PASSWORD=your-redis-password"
echo ""
echo "2. Run the application:"
echo "   go run ."
echo ""
echo "3. Test login/logout functionality in your browser"

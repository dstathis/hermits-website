#!/usr/bin/env bash
set -euo pipefail

# Run the full test suite with a temporary PostgreSQL container.
# Usage: ./scripts/test.sh

cd "$(dirname "$0")/.."

DC="docker compose -p hermits-test -f docker-compose.test.yml"

echo "==> Starting test database..."
$DC up -d db

echo "==> Waiting for database to be ready..."
for i in $(seq 1 30); do
    if $DC exec -T db pg_isready -U hermits_test >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

export TEST_DATABASE_URL="postgres://hermits_test:hermits_test@localhost:5433/hermits_test?sslmode=disable"

echo "==> Running tests..."
go test -v -count=1 -p 1 ./...
TEST_EXIT=$?

echo "==> Stopping test database..."
$DC down

exit $TEST_EXIT

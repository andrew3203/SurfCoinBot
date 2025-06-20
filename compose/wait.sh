#!/bin/sh
# Wait until PostgreSQL is ready before starting the bot

echo "⏳ Waiting for PostgreSQL at $POSTGRES_HOST:$POSTGRES_PORT..."

until pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER"; do
  sleep 1
done

echo "✅ PostgreSQL is ready"
exec "$@"

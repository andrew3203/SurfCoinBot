#!/bin/sh
set -e

DB_DSN="host=${POSTGRES_HOST} port=${POSTGRES_PORT} user=${POSTGRES_USER} password=${POSTGRES_PASSWORD} dbname=${POSTGRES_DB} sslmode=disable"

case "$1" in
  migrate)
    echo "🧱 Running DB migrations..."
    goose -dir ./migrations postgres "$DB_DSN" up
    ;;

  start)
    echo "🚀 Starting bot..."
    /wait.sh ./bot
    ;;

  *)
    echo "❌ Unknown command: $1"
    echo "Use: migrate | start"
    exit 1
    ;;
esac

#!/bin/sh
set -e

: "${DATABASE_URL:=postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}}"
export DATABASE_URL

envsubst < ./db/create-roles.template.sql > ./db/create-roles.sql

./migrate

psql "$DATABASE_URL" -f ./db/create-roles.sql

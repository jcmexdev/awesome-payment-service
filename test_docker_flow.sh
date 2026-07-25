#!/bin/bash

# Configuration
# Since we are running the seeder from the host, set the DATABASE_URL to target the PostgreSQL port mapped to your localhost.
export DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable}"

echo "================================================================================"
echo "=== 1. Preparando base de datos (Ejecutando Seeder desde el host)... ==="
echo "================================================================================"
go run cmd/seed-db/main.go

ACCOUNT_ID="test-account-123"
IDEMPOTENCY_KEY=$(uuidgen | tr '[:upper:]' '[:lower:]')

echo -e "\n================================================================================"
echo "=== 2. Enviando cargo exitoso de \$30.00 USD a través del API Gateway KrakenD ==="
echo "================================================================================"
curl -i -X POST http://localhost:8080/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d "{\"account_id\": \"$ACCOUNT_ID\", \"amount\": 3000, \"currency\": \"USD\"}"

echo -e "\n\n================================================================================"
echo "=== 3. Enviando mismo cargo (Debe retornar HIT-IDEMPOTENCY desde la caché) ==="
echo "================================================================================"
curl -i -X POST http://localhost:8080/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d "{\"account_id\": \"$ACCOUNT_ID\", \"amount\": 3000, \"currency\": \"USD\"}"

echo -e "\n\n================================================================================"
echo "=== 4. Enviando cargo de \$120.00 USD (Debe fallar por saldo insuficiente) ==="
echo "================================================================================"
INSUFFICIENT_KEY=$(uuidgen | tr '[:upper:]' '[:lower:]')
curl -i -X POST http://localhost:8080/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $INSUFFICIENT_KEY" \
  -d "{\"account_id\": \"$ACCOUNT_ID\", \"amount\": 12000, \"currency\": \"USD\"}"

echo -e "\n\n================================================================================"
echo "=== Proceso de prueba completado ==="
echo "================================================================================"

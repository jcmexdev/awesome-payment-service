#!/bin/bash

# Configuration
export DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable}"
export PORT="8080"
export REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"

echo "================================================================================"
echo "=== 1. Preparando base de datos (Ejecutando Seeder)... ==="
echo "================================================================================"
go run cmd/seed-db/main.go

echo -e "\n================================================================================"
echo "=== 2. Iniciando servidor de pagos en segundo plano... ==="
echo "================================================================================"
go run cmd/payments-api/main.go &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Verify server is up
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "ERROR: El servidor HTTP no pudo iniciar."
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

ACCOUNT_ID="test-account-123"
IDEMPOTENCY_KEY=$(uuidgen | tr '[:upper:]' '[:lower:]')

echo -e "\n================================================================================"
echo "=== 3. Enviando cargo exitoso de \$30.00 USD (3000 centavos) ==="
echo "================================================================================"
curl -i -X POST http://localhost:8080/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d "{\"account_id\": \"$ACCOUNT_ID\", \"amount\": 3000, \"currency\": \"USD\"}"

echo -e "\n\n================================================================================"
echo "=== 4. Enviando mismo cargo (Debe retornar HIT-IDEMPOTENCY desde la caché) ==="
echo "================================================================================"
curl -i -X POST http://localhost:8080/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -d "{\"account_id\": \"$ACCOUNT_ID\", \"amount\": 3000, \"currency\": \"USD\"}"

echo -e "\n\n================================================================================"
echo "=== 5. Enviando cargo de \$120.00 USD (Debe fallar por saldo insuficiente) ==="
echo "================================================================================"
INSUFFICIENT_KEY=$(uuidgen | tr '[:upper:]' '[:lower:]')
curl -i -X POST http://localhost:8080/v1/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $INSUFFICIENT_KEY" \
  -d "{\"account_id\": \"$ACCOUNT_ID\", \"amount\": 12000, \"currency\": \"USD\"}"

echo -e "\n\n================================================================================"
echo "=== 6. Apagando el servidor API de pagos... ==="
echo "================================================================================"
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null
echo "Servidor detenido con éxito."

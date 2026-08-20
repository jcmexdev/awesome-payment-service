package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/config"
	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/infra/cache/gorm"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	fmt.Println("Connecting to PostgreSQL at:", cfg.DatabaseURL)
	db, err := gorm.NewConnection(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to retrieve sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// 1. Seed a test Account
	accountID := "test-account-123"
	account := domain.Account{
		ID:            accountID,
		UserID:        "user-example-456",
		Currency:      "USD",
		CachedBalance: 10000, // $100.00 USD (in cents)
		Version:       0,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	fmt.Printf("Seeding test account '%s' with $100.00 USD (10000 cents)... ", accountID)
	// Upsert account
	var existing domain.Account
	err = db.Where("id = ?", accountID).First(&existing).Error
	if err != nil {
		err = db.Create(&account).Error
		if err != nil {
			log.Fatalf("Failed to create seed account: %v", err)
		}
		fmt.Println("Created!")
	} else {
		// Reset balance for fresh testing
		existing.CachedBalance = 10000
		existing.Version = 0
		db.Save(&existing)
		// Clean up any ledger entries or idempotency records for fresh testing
		db.Exec("DELETE FROM ledger_entries WHERE account_id = ?", accountID)
		db.Exec("DELETE FROM idempotency_records")
		fmt.Println("Reset and cleaned up database for fresh test run!")
	}

	// 2. Print instructions
	idempotencyKey := uuid.New().String()
	fmt.Println("\n================================================================================")
	fmt.Println("1. INICIAR EL SERVIDOR DE PAGOS:")
	fmt.Println("   DATABASE_URL=\""+cfg.DatabaseURL+"\" go run cmd/payments-api/main.go")
	fmt.Println("================================================================================")
	fmt.Println("2. PROBAR EL FLUJO CON CURL:")
	fmt.Println()
	fmt.Println("   a) Hacer un pago exitoso de $30.00 USD (3000 centavos):")
	fmt.Printf("      curl -i -X POST http://localhost:8080/v1/payments \\\n")
	fmt.Printf("        -H \"Content-Type: application/json\" \\\n")
	fmt.Printf("        -H \"Idempotency-Key: %s\" \\\n", idempotencyKey)
	fmt.Printf("        -d '{\"account_id\": \"%s\", \"amount\": 3000, \"currency\": \"USD\"}'\n", accountID)
	fmt.Println()
	fmt.Println("   b) Reintentar el mismo pago (Idempotencia - debe retornar la misma respuesta en caché):")
	fmt.Printf("      curl -i -X POST http://localhost:8080/v1/payments \\\n")
	fmt.Printf("        -H \"Content-Type: application/json\" \\\n")
	fmt.Printf("        -H \"Idempotency-Key: %s\" \\\n", idempotencyKey)
	fmt.Printf("        -d '{\"account_id\": \"%s\", \"amount\": 3000, \"currency\": \"USD\"}'\n", accountID)
	fmt.Println()
	fmt.Println("   c) Intentar pagar $120.00 USD (12000 centavos) - debe fallar por saldo insuficiente:")
	fmt.Printf("      curl -i -X POST http://localhost:8080/v1/payments \\\n")
	fmt.Printf("        -H \"Content-Type: application/json\" \\\n")
	fmt.Printf("        -H \"Idempotency-Key: %s\" \\\n", uuid.New().String())
	fmt.Printf("        -d '{\"account_id\": \"%s\", \"amount\": 12000, \"currency\": \"USD\"}'\n", accountID)
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("3. CONSULTAR EL ESTADO EN BASE DE DATOS (Mediante CLI o psql):")
	fmt.Printf("   - Cuenta:          SELECT * FROM accounts WHERE id = '%s';\n", accountID)
	fmt.Printf("   - Transacciones:   SELECT * FROM ledger_entries WHERE account_id = '%s';\n", accountID)
	fmt.Println("================================================================================")
}

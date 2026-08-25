package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/infra/http/consts"
)

type Account struct {
	ID            string    `json:"ID"`
	UserID        string    `json:"UserID"`
	Currency      string    `json:"Currency"`
	CachedBalance int64     `json:"Balance"`
	Version       int       `json:"Version"`
	CreatedAt     time.Time `json:"CreatedAt"`
	UpdatedAt     time.Time `json:"UpdatedAt"`
}

func main() {
	baseURL := getEnv("BASE_URL", "http://localhost:8080")
	numWorkers := 10
	targetAccountCount := 20

	fmt.Println("================================================================================")
	fmt.Printf("INICIANDO GENERADOR DE TRÁFICO Y CAOS EN: %s\n", baseURL)
	fmt.Printf("Hilos Concurrentes: %d | Cuentas Iniciales: %d\n", numWorkers, targetAccountCount)
	fmt.Println("================================================================================")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 1. Crear cuentas de prueba dynamically
	accounts := make([]string, 0, targetAccountCount)
	var accountsMu sync.Mutex

	fmt.Printf("Creando %d cuentas iniciales... \n", targetAccountCount)
	var wg sync.WaitGroup
	for i := 0; i < targetAccountCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			accID, err := createAccount(client, baseURL, fmt.Sprintf("load-user-%d", idx), 50000) // $500.00 USD
			if err != nil {
				fmt.Printf("  [!] Error creando cuenta %d: %v\n", idx, err)
				return
			}
			accountsMu.Lock()
			accounts = append(accounts, accID)
			accountsMu.Unlock()
		}(i)
	}
	wg.Wait()

	if len(accounts) == 0 {
		fmt.Println("[ERR] No se pudieron crear cuentas. Abortando generador de carga.")
		return
	}
	fmt.Printf("[OK] Se crearon %d cuentas de prueba exitosamente.\n", len(accounts))

	// 2. Generar carga concurrente
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var totalReqs int64
	var successReqs int64
	var errorReqs int64
	var statusCodes sync.Map

	for w := 0; w < numWorkers; w++ {
		go func(workerID int) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Elegir cuenta aleatoria
					accountsMu.Lock()
					accID := accounts[rand.Intn(len(accounts))]
					accountsMu.Unlock()

					// Generar pago
					amount := int64(100 + rand.Intn(1500)) // $1.00 a $16.00 USD
					err := sendPayment(client, baseURL, accID, amount, &totalReqs, &successReqs, &errorReqs, &statusCodes)
					if err != nil {
						// Conexión fallida o similar
					}

					// Pequeño retardo entre peticiones del worker (50-200ms)
					time.Sleep(time.Duration(50+rand.Intn(150)) * time.Millisecond)
				}
			}
		}(w)
	}

	// Reportero de estadísticas
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				total := atomic.LoadInt64(&totalReqs)
				success := atomic.LoadInt64(&successReqs)
				errorsCount := atomic.LoadInt64(&errorReqs)

				fmt.Printf("[%s] Reqs: %d | Éxitos: %d | Errores: %d\n",
					time.Now().Format("15:04:05"), total, success, errorsCount)
			}
		}
	}()

	<-ctx.Done()
	fmt.Println("\nDeteniendo generador de tráfico...")
	time.Sleep(1 * time.Second)
	fmt.Println("Generador detenido. Revisa tus dashboards en Grafana.")
}

func createAccount(client *http.Client, baseURL, userID string, balance int64) (string, error) {
	url := fmt.Sprintf("%s/v1/accounts", baseURL)
	body, _ := json.Marshal(map[string]interface{}{
		"user_id":  userID,
		"currency": "USD",
		"balance":  balance,
	})

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("http status %d: %s", resp.StatusCode, string(b))
	}

	var res struct {
		Data Account `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.Data.ID, nil
}

func sendPayment(client *http.Client, baseURL, accountID string, amount int64, total, success, errorsCount *int64, statusCodes *sync.Map) error {
	url := fmt.Sprintf("%s/v1/payments", baseURL)
	idempotencyKey := uuid.New().String()

	body, _ := json.Marshal(map[string]interface{}{
		"account_id": accountID,
		"amount":     amount,
		"currency":   "USD",
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	// INYECCIÓN DE CAOS ALEATORIA
	// 1. Simular latencia (5% de probabilidad)
	if rand.Float64() < 0.05 {
		req.Header.Set("X-Simulate-Delay", "500ms")
	}

	// 2. Simular fallos de base de datos / deadlocks (10% de probabilidad)
	if rand.Float64() < 0.10 {
		req.Header.Set("X-Simulate-Error", "true")
	}

	// 3. Simular peticiones erróneas para forzar códigos 400 y 422 (15% de probabilidad acumulada)
	roll := rand.Float64()
	if roll < 0.05 {
		// Enviar monto inválido para provocar un 400 Bad Request
		body, _ = json.Marshal(map[string]interface{}{
			"account_id": accountID,
			"amount":     -5,
			"currency":   "USD",
		})
		req, _ = http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)
	} else if roll < 0.10 {
		// Enviar account_id vacío para provocar un 400 Bad Request
		body, _ = json.Marshal(map[string]interface{}{
			"account_id": "",
			"amount":     amount,
			"currency":   "USD",
		})
		req, _ = http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)
	} else if roll < 0.15 {
		// Cuenta inexistente para provocar un 404 Not Found
		body, _ = json.Marshal(map[string]interface{}{
			"account_id": uuid.New().String(),
			"amount":     amount,
			"currency":   "USD",
		})
		req, _ = http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	atomic.AddInt64(total, 1)
	resp, err := client.Do(req)
	if err != nil {
		atomic.AddInt64(errorsCount, 1)
		return err
	}
	defer resp.Body.Close()

	// Contabilizar respuestas
	countVal, _ := statusCodes.LoadOrStore(resp.StatusCode, new(int64))
	atomic.AddInt64(countVal.(*int64), 1)

	if resp.StatusCode == http.StatusAccepted {
		atomic.AddInt64(success, 1)
	} else {
		atomic.AddInt64(errorsCount, 1)

		// Imprimir detalles de traza para depuración ante fallos
		reqID := resp.Header.Get(consts.HeaderRequestID)
		traceID := resp.Header.Get(consts.HeaderTraceID)
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("[GEN-FAIL] Status: %d | RequestID: %s | TraceID: %s | Response: %s\n",
			resp.StatusCode, reqID, traceID, string(bodyBytes))
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

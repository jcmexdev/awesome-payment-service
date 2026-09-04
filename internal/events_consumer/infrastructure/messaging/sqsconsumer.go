package messaging

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/jcmexdev/payment-service/internal/events_consumer/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type SQSWorker struct {
	client    *sqs.Client
	queueURL  string
	processor ports.PaymentProcessorUseCase
	tracer    trace.Tracer
}

func NewSQSWorker(sqsClient *sqs.Client, queueURL string, processor ports.PaymentProcessorUseCase, tracer trace.Tracer) *SQSWorker {
	return &SQSWorker{client: sqsClient, queueURL: queueURL, processor: processor, tracer: tracer}
}

func (w *SQSWorker) Start(ctx context.Context, workers int) error {
	log.Printf("[SQSWorker] Iniciando %d workers en la cola %s...", workers, w.queueURL)
	jobs := make(chan types.Message, workers*2)
	var wg sync.WaitGroup

	// 1. Levantar el Pool de Goroutines (Workers)
	for i := 1; i <= workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.runWorker(ctx, workerID, jobs)
		}(i)
	}

	// 2. Dispatcher Loop: Realiza Long Polling e inserta mensajes en el canal
	go func() {
		defer close(jobs)
		for {
			select {
			case <-ctx.Done():
				log.Println("[SQSWorker] Deteniendo el loop de polling...")
				return
			default:
				// Long Polling (WaitTimeSeconds = 20)
				out, err := w.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
					QueueUrl:            aws.String(w.queueURL),
					MaxNumberOfMessages: 10, // Procesa lotes de hasta 10 mensajes
					WaitTimeSeconds:     20, // Long Polling activa
					VisibilityTimeout:   30, // Tiempo reservado para procesar
					MessageAttributeNames: []string{
						".*", // Solicitar todas las cabeceras (incluyendo traceparent y event_id)
					},
				})

				if err != nil {
					// Prevenir un bucle rápido e ininterrumpido en caso de falla de red/AWS
					if ctx.Err() == nil {
						log.Printf("[SQSWorker] Error en ReceiveMessage: %v", err)
						time.Sleep(2 * time.Second)
					}
					continue
				}

				if len(out.Messages) == 0 {
					log.Printf("[SQSWorker] No Messages found in queue: %s", w.queueURL)
				}

				for _, msg := range out.Messages {
					select {
					case jobs <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	wg.Wait()
	log.Println("[SQSWorker] Todos los workers finalizados de forma segura.")
	return nil
}

func (w *SQSWorker) runWorker(ctx context.Context, id int, jobs <-chan types.Message) {
	for msg := range jobs {
		w.processSingleMessage(ctx, msg)
	}
}

// processSingleMessage procesa un único mensaje de SQS con su traza y confirmación (ACK)
func (w *SQSWorker) processSingleMessage(ctx context.Context, msg types.Message) {
	// 1. Convertir cabeceras de SQS para OpenTelemetry
	headers := extractHeaders(msg.MessageAttributes)

	// 2. Extraer el contexto de la traza distribuida (Trace Propagation)
	parentCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))

	// 3. Crear el Span OTel local para el procesamiento del mensaje
	ctx, span := w.tracer.Start(parentCtx, "SQSWorker.ProcessMessage")
	defer span.End()

	eventID := headers["event_id"]

	if msg.Body == nil {
		log.Printf("[SQSWorker] Mensaje sin cuerpo recibido (MessageId: %s)", aws.ToString(msg.MessageId))
		w.deleteMessage(ctx, msg.ReceiptHandle) // Eliminar para no procesar mensajes vacíos inválidos
		return
	}

	payload := []byte(*msg.Body)

	// 4. Invocación de la capa de aplicación (Caso de Uso)
	err := w.processor.ProcessTransaction(ctx, payload)
	if err != nil {
		log.Printf("[SQSWorker] Error procesando evento %s: %v. Dejando expirar VisibilityTimeout.", eventID, err)
		// NO borramos el mensaje: SQS lo reintentará tras expirar el VisibilityTimeout
		return
	}

	// 5. Confirmación Exitosa (ACK / DeleteMessage)
	w.deleteMessage(ctx, msg.ReceiptHandle)
}

// deleteMessage elimina el mensaje de la cola de SQS
func (w *SQSWorker) deleteMessage(ctx context.Context, receiptHandle *string) {
	_, err := w.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(w.queueURL),
		ReceiptHandle: receiptHandle,
	})
	if err != nil {
		log.Printf("[SQSWorker] Error haciendo DeleteMessage (ACK): %v", err)
	}
}

// extractHeaders convierte MessageAttributes a map[string]string compatible con OTel
func extractHeaders(attrs map[string]types.MessageAttributeValue) map[string]string {
	headers := make(map[string]string, len(attrs))
	for k, v := range attrs {
		if v.StringValue != nil {
			headers[k] = *v.StringValue
		}
	}
	return headers
}

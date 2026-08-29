package messaging_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/infrastructure/messaging"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/ports"
)

type mockSQSClient struct {
	sentInput *sqs.SendMessageInput
}

func (m *mockSQSClient) GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
	return &sqs.GetQueueUrlOutput{
		QueueUrl: aws.String("https://sqs.us-east-1.amazonaws.com/123456789012/payments-queue"),
	}, nil
}

func (m *mockSQSClient) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	m.sentInput = params
	return &sqs.SendMessageOutput{
		MessageId: aws.String("msg-id-12345"),
	}, nil
}

func TestSQSPublisher_Publish_SendsTraceDataInMessageAttributes(t *testing.T) {
	mockClient := &mockSQSClient{}
	publisher, err := messaging.NewSQSPublisher(context.Background(), mockClient, "payments-queue")
	if err != nil {
		t.Fatalf("unexpected error creating SQSPublisher: %v", err)
	}

	expectedTraceParent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	expectedBaggage := "userId=user-123"

	msg := ports.Message{
		Destination: "payments-queue",
		Key:         "pay-123",
		Payload:     []byte(`{"payment_id":"pay-123","amount":5000}`),
		Headers: map[string]string{
			"traceparent": expectedTraceParent,
			"baggage":     expectedBaggage,
		},
	}

	err = publisher.Publish(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error publishing message: %v", err)
	}

	// 1. Validar que SendMessage recibió el input
	if mockClient.sentInput == nil {
		t.Fatal("expected SendMessage to be called on SQS client")
	}

	input := mockClient.sentInput

	// 2. Validar payload del mensaje
	if aws.ToString(input.MessageBody) != string(msg.Payload) {
		t.Errorf("expected MessageBody %s, got %s", string(msg.Payload), aws.ToString(input.MessageBody))
	}

	// 3. Validar que MessageAttributes contiene los datos de trace
	if input.MessageAttributes == nil {
		t.Fatal("expected MessageAttributes to be populated in SQS message, got nil")
	}

	// Validar traceparent
	traceAttr, exists := input.MessageAttributes["traceparent"]
	if !exists {
		t.Fatal("expected 'traceparent' attribute in SQS MessageAttributes")
	}
	if aws.ToString(traceAttr.DataType) != "String" {
		t.Errorf("expected DataType 'String', got %s", aws.ToString(traceAttr.DataType))
	}
	if aws.ToString(traceAttr.StringValue) != expectedTraceParent {
		t.Errorf("expected traceparent value %s, got %s", expectedTraceParent, aws.ToString(traceAttr.StringValue))
	}

	// Validar baggage
	baggageAttr, exists := input.MessageAttributes["baggage"]
	if !exists {
		t.Fatal("expected 'baggage' attribute in SQS MessageAttributes")
	}
	if aws.ToString(baggageAttr.StringValue) != expectedBaggage {
		t.Errorf("expected baggage value %s, got %s", expectedBaggage, aws.ToString(baggageAttr.StringValue))
	}
}

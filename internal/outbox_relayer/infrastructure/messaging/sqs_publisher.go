package messaging

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/ports"
)

type SQSPublisher struct {
	client   *sqs.Client
	queueURL string
}

// Ensure interface implementation compile check
var _ ports.MessagePublisher = (*SQSPublisher)(nil)

func NewSQSPublisher(ctx context.Context, client *sqs.Client, queueName string) (*SQSPublisher, error) {
	result, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, fmt.Errorf("error obteniendo URL de la cola %s: %w", queueName, err)
	}
	fmt.Println(&result.QueueUrl)
	return &SQSPublisher{client: client, queueURL: *result.QueueUrl}, nil
}

func (s *SQSPublisher) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(payload)),
	}

	_, err := s.client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("error publicando mensaje en SQS (%s): %w", topic, err)
	}
	fmt.Println("Successfully published a message")
	return nil
}

func (s *SQSPublisher) PublishV1(ctx context.Context, msg ports.Message) error {
	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(msg.Payload)),
	}

	_, err := s.client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("error publicando mensaje en SQS (%s): %w", s.queueURL, err)
	}

	fmt.Println("Successfully published a message")
	return nil
}

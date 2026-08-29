package messaging

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/ports"
)

type SQSClientAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
}

type SQSPublisher struct {
	client   SQSClientAPI
	queueURL string
}

var _ ports.MessagePublisher = (*SQSPublisher)(nil)

func NewSQSPublisher(ctx context.Context, client SQSClientAPI, queueName string) (*SQSPublisher, error) {
	result, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, fmt.Errorf("error obteniendo URL de la cola %s: %w", queueName, err)
	}
	return &SQSPublisher{client: client, queueURL: *result.QueueUrl}, nil
}

func (s *SQSPublisher) Publish(ctx context.Context, msg ports.Message) error {
	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(msg.Payload)),
	}

	if len(msg.Headers) > 0 {
		msgAttributes := make(map[string]types.MessageAttributeValue, len(msg.Headers))
		for k, v := range msg.Headers {
			msgAttributes[k] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(v),
			}
		}
		input.MessageAttributes = msgAttributes
	}

	_, err := s.client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("error publicando mensaje en SQS (%s): %w", s.queueURL, err)
	}

	fmt.Println("Successfully published a message")
	return nil
}

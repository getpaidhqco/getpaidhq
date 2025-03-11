package sqs

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"payloop/internal/application/lib/events"
	"payloop/internal/lib"

	"payloop/internal/application/lib/logger"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSFifoClient struct {
	client *sqs.Client
	logger logger.Logger
	env    lib.Env
}

func NewSQSFifoClient(logger logger.Logger, env lib.Env) events.QueueClient {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(env.Get("AWS_REGION")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			env.Get("SQS_ACCESS_KEY_ID"),
			env.Get("SQS_SECRET_ACCESS_KEY"),
			"",
		)),
	)

	if err != nil {
		panic(err)
	}

	queueUrl := env.Get("SQS_QUEUE_URL")
	if queueUrl == "" {
		panic("SQS_QUEUE_URL not set")
	}

	client := sqs.NewFromConfig(cfg)

	return SQSFifoClient{
		logger: logger,
		env:    env,
		client: client,
	}
}

func (c SQSFifoClient) Start(handler events.QueueMessageHandler) {
	queueUrl := c.env.Get("SQS_QUEUE_URL")
	c.logger.Infof("Starting SQS FIFO client for queue [%s]", queueUrl)
	go func() {
		for {
			// Receive messages from the queue
			msgResult, err := c.client.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(queueUrl),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     20,
				MessageAttributeNames: []string{
					string(types.QueueAttributeNameAll),
				},
			})
			if err != nil {
				c.logger.Errorf("failed to receive messages, %v", err)
				time.Sleep(30 * time.Second)
				continue
			}

			if len(msgResult.Messages) == 0 {
				continue
			}

			for _, msg := range msgResult.Messages {
				// Process the message
				c.logger.Debugf("[SQSFifoClient] Processing message [%s]", aws.ToString(msg.MessageId))
				var queueMessage events.QueueMessage
				err := json.Unmarshal([]byte(*msg.Body), &queueMessage)
				if err != nil {
					c.logger.Errorf("[SQSFifoClient] failed to unmarshal queue message: %v", err)
					_ = c.deleteMessage(aws.ToString(msg.ReceiptHandle))
					continue
				}

				err = handler(queueMessage)
				if err != nil {
					if !events.IsRetryable(err) {
						c.logger.Errorf("non-retryable error %s, %v", aws.ToString(msg.MessageId), err.Error())
						_ = c.deleteMessage(aws.ToString(msg.ReceiptHandle))
					}
					continue
				}

				// Processing was successful, delete the message after processing
				_, err = c.client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(queueUrl),
					ReceiptHandle: msg.ReceiptHandle,
				})
				if err != nil {
					c.logger.Errorf("failed to delete message %s, %v", aws.ToString(msg.MessageId), err)
				}
			}

			// Sleep for a while before polling again
			time.Sleep(5 * time.Second)
		}
	}()
}

func (c SQSFifoClient) SendMessage(ctx context.Context, data events.QueueMessage) error {
	queueUrl := c.env.Get("SQS_QUEUE_URL")
	if queueUrl == "" {
		return lib.NewCustomError(lib.InternalError, "SQS_QUEUE_URL not set", nil)
	}

	messageBody, err := json.Marshal(data)
	if err != nil {
		return lib.NewCustomError(lib.InternalError, "failed to marshal message data", err)
	}

	_, err = c.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:       aws.String(queueUrl),
		MessageBody:    aws.String(string(messageBody)),
		MessageGroupId: aws.String("payloop"),
	})
	if err != nil {
		return lib.NewCustomError(lib.InternalError, "failed to send message", err)
	}

	return nil
}

func (c SQSFifoClient) deleteMessage(id string) error {
	queueUrl := c.env.Get("SQS_QUEUE_URL")
	// Delete the message after processing
	_, err := c.client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueUrl),
		ReceiptHandle: aws.String(id),
	})
	if err != nil {
		c.logger.Errorf("failed to delete message %s, %v", id, err)
	}
	return err
}

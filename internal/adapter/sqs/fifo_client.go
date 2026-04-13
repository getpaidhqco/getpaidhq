package sqs

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"payloop/internal/core/port"
	"payloop/internal/lib"

	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSFifoClient struct {
	client *sqs.Client
	logger port.Logger
	env    lib.Env
}

func NewSQSFifoClient(logger port.Logger, env lib.Env) port.QueueClient {
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

func (c SQSFifoClient) Start(handler port.QueueMessageHandler) {
	queueUrl := c.env.Get("SQS_QUEUE_URL")
	c.logger.Info("starting sqs fifo client", "queueUrl", queueUrl)
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
				c.logger.Error("failed to receive messages", "error", err)
				time.Sleep(30 * time.Second)
				continue
			}

			if len(msgResult.Messages) == 0 {
				continue
			}

			c.logger.Debug("processing messages", "count", len(msgResult.Messages))
			for _, msg := range msgResult.Messages {
				start := time.Now()
				// Process the message
				c.logger.Debug("processing message", "messageId", aws.ToString(msg.MessageId))
				var queueMessage port.QueueMessage
				err := json.Unmarshal([]byte(*msg.Body), &queueMessage)
				if err != nil {
					c.logger.Error("failed to unmarshal queue message", "error", err)
					_ = c.deleteMessage(aws.ToString(msg.ReceiptHandle))
					continue
				}

				err = handler(queueMessage)
				if err != nil {
					if !port.IsRetryable(err) {
						c.logger.Error("non-retryable error", "messageId", aws.ToString(msg.MessageId), "error", err)
						_ = c.deleteMessage(aws.ToString(msg.ReceiptHandle))
					}
					continue
				}

				elapsed := time.Since(start)
				c.logger.Debug("processed message", "messageId", aws.ToString(msg.MessageId), "elapsed", elapsed)

				// Processing was successful, delete the message after processing
				_ = c.deleteMessage(aws.ToString(msg.ReceiptHandle))
				if err != nil {
					c.logger.Error("failed to delete message", "messageId", aws.ToString(msg.MessageId), "error", err)
				}
			}
		}
	}()
}

func (c SQSFifoClient) SendMessage(ctx context.Context, data port.QueueMessage) error {
	queueUrl := c.env.Get("SQS_QUEUE_URL")
	if queueUrl == "" {
		return lib.NewCustomError(lib.InternalError, "SQS_QUEUE_URL not set", nil)
	}

	messageBody, err := json.Marshal(data)
	if err != nil {
		return lib.NewCustomError(lib.InternalError, "failed to marshal message data", err)
	}

	rsp, err := c.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:       aws.String(queueUrl),
		MessageBody:    aws.String(string(messageBody)),
		MessageGroupId: aws.String("payloop"),
	})
	if err != nil {
		return lib.NewCustomError(lib.InternalError, "failed to send message", err)
	}

	c.logger.Debug("sent message to sqs fifo queue", "queueUrl", queueUrl, "messageId", *rsp.MessageId)
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
		c.logger.Error("failed to delete message", "id", id, "error", err)
	}
	return err
}

package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/fx"
	"payloop/internal/lib"
)

func Module() fx.Option {
	return fx.Module("s3-storage",
		fx.Provide(
			NewS3Client,
			NewS3Storage,
		),
	)
}

func NewS3Client(env lib.Env) (*s3.Client, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(env.S3Region),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(awsCfg), nil
}

func NewS3Storage(client *s3.Client, env lib.Env) *S3Storage {
	return &S3Storage{
		client: client,
		bucket: env.S3Bucket,
		region: env.S3Region,
	}
}

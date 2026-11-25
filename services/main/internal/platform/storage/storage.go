package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	ErrObjectNotFound = errors.New("object not found")
	ErrUnavailable    = errors.New("object storage is unavailable")
)

type Object struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

type Client interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Get(ctx context.Context, key string) (*Object, error)
	Delete(ctx context.Context, key string) error
}

type Settings struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

type S3Client struct {
	client *s3.Client
	bucket string
}

func NewS3Client(ctx context.Context, settings Settings) (*S3Client, error) {
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(settings.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(settings.AccessKeyID, settings.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load object storage configuration: %w", err)
	}

	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.UsePathStyle = true
		endpointIsConfigured := settings.Endpoint != ""
		if endpointIsConfigured {
			options.BaseEndpoint = aws.String(settings.Endpoint)
		}
	})
	return &S3Client{client: client, bucket: settings.Bucket}, nil
}

func (c *S3Client) Put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Body:        bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("%w: put %s: %v", ErrUnavailable, key, err)
	}
	return nil
}

func (c *S3Client) Get(ctx context.Context, key string) (*Object, error) {
	output, err := c.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(c.bucket), Key: aws.String(key)})
	if err != nil {
		var missing *types.NoSuchKey
		if errors.As(err, &missing) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("%w: get %s: %v", ErrUnavailable, key, err)
	}

	return &Object{
		Body:        output.Body,
		ContentType: aws.ToString(output.ContentType),
		Size:        aws.ToInt64(output.ContentLength),
	}, nil
}

func (c *S3Client) Delete(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(c.bucket), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("%w: delete %s: %v", ErrUnavailable, key, err)
	}
	return nil
}

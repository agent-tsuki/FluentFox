// Package storage — Cloudflare R2 implementation of the Storage interface.
// This is the only file in the codebase that imports aws-sdk-go-v2.
// All other packages consume the Storage interface.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Store implements Storage backed by Cloudflare R2 via the S3-compatible API.
type R2Store struct {
	client     *s3.Client
	presigner  *s3.PresignClient
	bucket     string
	publicURL  string
}

// NewR2Store constructs an R2Store. accountID, accessKey, secretKey, bucket,
// and publicURL all come from Config — never from env directly.
func NewR2Store(accountID, accessKey, secretKey, bucket, publicURL string) *R2Store {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg := aws.Config{
		Region: "auto",
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(
			func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint}, nil
			},
		),
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &R2Store{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    bucket,
		publicURL: publicURL,
	}
}

// GeneratePresignedUpload returns a time-limited PUT URL for direct client uploads.
func (s *R2Store) GeneratePresignedUpload(ctx context.Context, key, contentType string, expiresInSecs int) (string, error) {
	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(time.Duration(expiresInSecs)*time.Second))
	if err != nil {
		return "", fmt.Errorf("storage: presign upload for key %q: %w", key, err)
	}
	return req.URL, nil
}

// GeneratePresignedDownload returns a time-limited GET URL for private object access.
func (s *R2Store) GeneratePresignedDownload(ctx context.Context, key string, expiresInSecs int) (string, error) {
	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(time.Duration(expiresInSecs)*time.Second))
	if err != nil {
		return "", fmt.Errorf("storage: presign download for key %q: %w", key, err)
	}
	return req.URL, nil
}

// Delete removes an object from the R2 bucket.
func (s *R2Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: delete key %q: %w", key, err)
	}
	return nil
}

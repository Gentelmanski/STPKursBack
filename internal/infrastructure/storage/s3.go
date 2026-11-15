package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"auth-system/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(cfg *config.Config) (*S3, error) {
	if cfg.S3Bucket == "" || cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
		return nil, errors.New("incomplete S3 configuration")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 10 * time.Second
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure:       cfg.S3UseSSL,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
		Transport:    transport,
		MaxRetries:   2,
	})
	if err != nil {
		return nil, errors.New("invalid S3 configuration")
	}
	return &S3{client: client, bucket: cfg.S3Bucket}, nil
}

func (s *S3) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
		// A file is at most 5 MiB: a single PUT avoids abandoned multipart uploads.
		DisableMultipart: true,
	})
	return err
}

func (s *S3) Remove(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if minio.ToErrorResponse(err).Code == "NoSuchKey" {
		return nil
	}
	return err
}

func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	params := make(url.Values)
	params.Set("response-content-disposition", "inline")
	params.Set("response-cache-control", "private, no-store")
	signed, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, params)
	if err != nil {
		return "", err
	}
	return signed.String(), nil
}

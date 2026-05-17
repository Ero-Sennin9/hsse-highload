package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"ads-service/internal/domain/ports"
)

var _ ports.ObjectStorage = (*ObjectStorage)(nil)

type ObjectStorage struct {
	client     *minio.Client
	bucket     string
	publicBase string
}

func NewObjectStorage() (*ObjectStorage, error) {
	endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	accessKey := strings.TrimSpace(os.Getenv("S3_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("S3_SECRET_KEY"))
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	publicBase := strings.TrimSpace(os.Getenv("MEDIA_PUBLIC_BASE_URL"))
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET required")
	}
	if publicBase == "" {
		publicBase = fmt.Sprintf("http://%s/%s", endpoint, bucket)
	}
	useSSL, _ := strconv.ParseBool(os.Getenv("S3_USE_SSL"))
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &ObjectStorage{
		client:     client,
		bucket:     bucket,
		publicBase: strings.TrimRight(publicBase, "/"),
	}, nil
}

func (s *ObjectStorage) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *ObjectStorage) PublicURL(key string) string {
	return s.publicBase + "/" + strings.TrimPrefix(key, "/")
}

package objectstorage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

const (
	DefaultURLExpireSeconds = 900
	MaxURLExpireSeconds     = 604800
)

type Config struct {
	Enabled          bool   `json:"enabled"`
	Endpoint         string `json:"endpoint"`
	Region           string `json:"region"`
	Bucket           string `json:"bucket"`
	AccessKey        string `json:"access_key"`
	SecretKey        string `json:"secret_key"`
	KeyPrefix        string `json:"key_prefix"`
	URLExpireSeconds int64  `json:"url_expire_seconds"`
	PublicBaseURL    string `json:"public_base_url"`
}

type Object struct {
	Key string
	URL string
}

type Storage interface {
	Put(ctx context.Context, key string, content io.Reader, contentLength int64, contentType string) error
	PresignGet(key string) (string, error)
	Delete(ctx context.Context, key string) error
}

type TOS struct {
	client *tos.ClientV2
	config Config
}

func NewTOS(config Config) (*TOS, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}
	client, err := tos.NewClientV2(config.Endpoint, tos.WithRegion(config.Region), tos.WithCredentials(tos.NewStaticCredentials(config.AccessKey, config.SecretKey)))
	if err != nil {
		return nil, fmt.Errorf("create TOS client: %w", err)
	}
	return &TOS{client: client, config: config}, nil
}

func ValidateConfig(config Config) error {
	if !config.Enabled {
		return nil
	}
	if config.Endpoint == "" || !strings.HasPrefix(strings.ToLower(config.Endpoint), "https://") {
		return fmt.Errorf("TOS endpoint must be an HTTPS URL")
	}
	if config.Region == "" || config.Bucket == "" || config.AccessKey == "" || config.SecretKey == "" {
		return fmt.Errorf("TOS region, bucket, access key and secret key are required")
	}
	if strings.ContainsAny(config.Bucket+config.KeyPrefix, "\r\n") {
		return fmt.Errorf("TOS bucket and key prefix contain invalid characters")
	}
	if config.URLExpireSeconds == 0 {
		config.URLExpireSeconds = DefaultURLExpireSeconds
	}
	if config.URLExpireSeconds < 1 || config.URLExpireSeconds > MaxURLExpireSeconds {
		return fmt.Errorf("TOS URL expiry must be between 1 and %d seconds", MaxURLExpireSeconds)
	}
	if config.PublicBaseURL != "" && !strings.HasPrefix(strings.ToLower(config.PublicBaseURL), "https://") {
		return fmt.Errorf("TOS public base URL must be HTTPS")
	}
	return nil
}

func (s *TOS) Config() Config { return s.config }

func (s *TOS) Put(ctx context.Context, key string, content io.Reader, contentLength int64, contentType string) error {
	if key == "" || path.IsAbs(key) || strings.Contains(key, "..") {
		return fmt.Errorf("invalid TOS object key")
	}
	_, err := s.client.PutObjectV2(ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket:        s.config.Bucket,
			Key:           key,
			ContentLength: contentLength,
			ContentType:   contentType,
		},
		Content: content,
	})
	return err
}

func (s *TOS) PresignGet(key string) (string, error) {
	out, err := s.client.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodGet,
		Bucket:     s.config.Bucket,
		Key:        key,
		Expires:    s.config.URLExpireSeconds,
	})
	if err != nil {
		return "", err
	}
	if s.config.PublicBaseURL == "" {
		return out.SignedUrl, nil
	}
	base, err := url.Parse(strings.TrimRight(s.config.PublicBaseURL, "/"))
	if err != nil {
		return "", err
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(key, "/")
	return base.String(), nil
}

func (s *TOS) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObjectV2(ctx, &tos.DeleteObjectV2Input{Bucket: s.config.Bucket, Key: key})
	return err
}

func BuildKey(prefix, extension string) string {
	return strings.Trim(strings.TrimSpace(prefix), "/") + "/" + time.Now().UTC().Format("20060102") + "/" + uuid.New().String() + extension
}

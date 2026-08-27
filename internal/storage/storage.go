package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const defaultRegion = "garage"

type Config struct {
	Endpoint      string
	Region        string
	AccessKey     string
	SecretKey     string
	Bucket        string
	PublicBaseURL string
	UseSSL        bool
}

type Client struct {
	s3         *s3.Client
	bucket     string
	publicBase string
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	if u, err := url.Parse(cfg.Endpoint); err == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https") {
		endpoint = u.Scheme + "://" + u.Host
	} else {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		endpoint = scheme + "://" + endpoint
	}

	region := cfg.Region
	if region == "" {
		region = defaultRegion
	}

	s3c := s3.New(s3.Options{
		Region:       region,
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		UsePathStyle: true,
	})

	if _, err := s3c.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(cfg.Bucket)}); err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return nil, fmt.Errorf("garage bucket %q does not exist", cfg.Bucket)
		}
		return nil, fmt.Errorf("garage bucket check: %w", err)
	}

	return &Client{
		s3:         s3c,
		bucket:     cfg.Bucket,
		publicBase: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

func (c *Client) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(c.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

func (c *Client) RemovePrefix(ctx context.Context, prefix string) error {
	pager := s3.NewListObjectsV2Paginator(c.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list %s: %w", prefix, err)
		}
		if len(page.Contents) == 0 {
			continue
		}

		ids := make([]types.ObjectIdentifier, len(page.Contents))
		for i, obj := range page.Contents {
			ids[i] = types.ObjectIdentifier{Key: obj.Key}
		}

		out, err := c.s3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(c.bucket),
			Delete: &types.Delete{Objects: ids, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("remove %s: %w", prefix, err)
		}
		if len(out.Errors) > 0 {
			return fmt.Errorf("remove %s: %s", aws.ToString(out.Errors[0].Key), aws.ToString(out.Errors[0].Message))
		}
	}
	return nil
}

// URL returns the public address of a stored object. PublicBaseURL is expected
// to be Garage's website endpoint for this bucket (bucket selected by Host), so
// the bucket name is not part of the path.
func (c *Client) URL(key string) string {
	return c.publicBase + "/" + key
}

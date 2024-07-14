package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/synchthia/packy/models"
)

type R2Client struct {
	directory string
	cacheSvc  *CacheService

	BucketName      string
	AccountID       string
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
}

func InitR2FromEnv(directory string, cacheSvc *CacheService) *R2Client {
	return &R2Client{
		directory: directory,
		cacheSvc:  cacheSvc,
		BucketName: func() string {
			if value := os.Getenv("R2_BUCKET_NAME"); len(value) >= 1 {
				return value
			}
			panic("R2_BUCKET_NAME not defined")
		}(),
		AccountID: func() string {
			if value := os.Getenv("R2_ACCOUNT_ID"); len(value) >= 1 {
				return value
			}
			panic("R2_ACCOUNT_ID not defined")
		}(),
		Endpoint: func() string {
			if value := os.Getenv("R2_ENDPOINT"); len(value) >= 1 {
				return value
			}
			panic("R2_ENDPOINT not defined")
		}(),
		AccessKeyID: func() string {
			if value := os.Getenv("R2_ACCESS_KEY_ID"); len(value) >= 1 {
				return value
			}
			panic("R2_ACCESS_KEY_ID not defined")
		}(),
		AccessKeySecret: func() string {
			if value := os.Getenv("R2_ACCESS_KEY_SECRET"); len(value) >= 1 {
				return value
			}
			panic("R2_ACCESS_KEY_SECRET not defined")
		}(),
	}
}

func (r2 *R2Client) getClient() (*s3.Client, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", r2.AccountID),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(r2.AccessKeyID, r2.AccessKeySecret, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg), nil
}

func (r2 *R2Client) List(namespace string) ([]*models.Content, error) {
	client, err := r2.getClient()
	if err != nil {
		return nil, err
	}

	listObjectsOutput, err := client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: &r2.BucketName,
		Prefix: aws.String(namespace),
	})
	if err != nil {
		return nil, err
	}

	files := []*models.Content{}

	for _, object := range listObjectsOutput.Contents {
		files = append(files, &models.Content{
			Name: func() string {
				value := strings.Split(*object.Key, "/")
				return value[len(value)-1]
			}(),
			Path: *object.Key,
			Hash: *object.ETag,
		})
	}

	return files, nil
}

func (r2 *R2Client) Fetch(namespace string, contents []*models.Content) error {
	client, err := r2.getClient()
	if err != nil {
		return err
	}

	fmt.Printf("[%s] fetching %d files...\n", namespace, len(contents))

	for _, content := range contents {
		cacheEntry := r2.cacheSvc.Cache.Files[content.Name]

		if cacheEntry == nil {
			r2.cacheSvc.Cache.Files[content.Name] = &CachedFile{
				Hash: content.Hash,
			}
			if err := r2.cacheSvc.Save(); err != nil {
				return err
			}
		} else {
			if cacheEntry.Hash == content.Hash {
				fmt.Printf("[%s] !! Skipped: %s\n", namespace, content.Name)
				continue
			}
		}

		f, err := client.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: &r2.BucketName,
			Key:    &content.Path,
		})
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(f.Body)

		file, err := os.Create(r2.directory + "/" + content.Name)
		if err != nil {
			return err
		}

		_, err = file.ReadFrom(f.Body)
		if err != nil {
			panic(err)
		}

		_, err = file.Write(buf.Bytes())
		if err != nil {
			return err
		}
		fmt.Printf("[%s] :: Downloaded: %s\n", namespace, content.Name)
	}

	return nil
}

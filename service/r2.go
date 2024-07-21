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

func (r2 *R2Client) Fetch(contents []*models.Content) error {
	client, err := r2.getClient()
	if err != nil {
		return err
	}

	// キャッシュにあるけど、R2に存在していないファイルならそのファイルは削除する
	for cachedFileName := range r2.cacheSvc.Cache.Files {
		if !isExists(cachedFileName, contents) {
			if _, err := os.Stat(r2.directory + "/" + cachedFileName); os.IsNotExist(err) {
				continue
			}
			fmt.Printf("!! Removed: %s\n", cachedFileName)
			if err := os.Remove(r2.directory + "/" + cachedFileName); err != nil {
				return err
			} else {
				delete(r2.cacheSvc.Cache.Files, cachedFileName)
			}
		}
	}

	fmt.Printf("==> Fetching %d files...\n", len(contents))
	for _, content := range contents {
		cacheEntry := r2.cacheSvc.Cache.Files[content.Name]
		if cacheEntry == nil {
			r2.cacheSvc.Cache.Files[content.Name] = &CachedFile{
				Hash: content.Hash,
			}
		} else {
			// ハッシュが同一な場合はスキップ
			if cacheEntry.Hash == content.Hash {
				fmt.Printf("!! Skipped: %s\n", content.Path)
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
			return err
		}

		_, err = file.Write(buf.Bytes())
		if err != nil {
			return err
		}
		fmt.Printf(":: Downloaded: %s\n", content.Path)
	}

	if err := r2.cacheSvc.Save(); err != nil {
		return err
	}

	return nil
}

func isExists(fileName string, contents []*models.Content) bool {
	for _, content := range contents {
		if content.Name == fileName {
			return true
		}
	}

	return false
}

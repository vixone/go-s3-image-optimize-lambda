package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/disintegration/imaging"
)

type S3Client struct {
	client    *s3.Client
	srcBucket string
	dstBucket string
}

func News3Client(srcBucket, dstBucket string) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		return nil, err, fmt.Errorf("unable to load AWS config, %v", err)
	}

	return &S3Client{
		client:    s3.NewFromConfig(cfg),
		srcBucket: srcBucket,
		dstBucket: dstBucket,
	}, nil
}

func (s *S3Client) ListImages(ctx context.Context) ([]string, error) {
	var imageKeys []string
	paginator := s.client.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: &s.srcBucket,
		Prefix: "uuid/",
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err, fmt.Errorf("Error listing objects in bucket, %v", err)
		}

		for _, object := range page.Contents {
			if object.Size > 0 { // only get non-folders
				imageKeys = append(imageKeys, *object.Key)
			}
		}
	}

	return imageKeys, nil
}

func (s *S3Client) GetImage(ctx context.Context, key string) ([]byte, error) {
	getObjectOutput, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.srcBucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err, fmt.Errorf("Error getting object from bucket, %v", err)
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(getObjectOutput.Body)
	if err != nil {
		return nil, err, fmt.Errorf("Error reading object from bucket, %v", err)
	}
	return buf.Bytes(), nil
}

func (s *S3Client) DownloadImage(ctx context.Context, key string) ([]byte, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.srcBucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %v", key, err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", key, err)
	}
	return buf.Bytes(), nil
}

func (s *S3Client) UploadImage(ctx context.Context, key string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.dstBucket,
		Key:    &key,
		Body:   bytes.NewReader(data),
	})
	return err
}

func OptimizeImage(imageData []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	// make width 800 and keep axpect ratio
	resized := imaging.Resize(img, 800, 0, imaging.Lanczos)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 80})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %v", err)
	}
	return buf.Bytes(), nil
}

func Worker(ctx context.Context, s3Client *S3Client, jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for key := range jobs {
		fmt.Println("Processing:", key)

		imgData, err := s3Client.DownloadImage(ctx, key)
		if err != nil {
			log.Println("Error downloading:", key, err)
			continue
		}

		optimizedData, err := OptimizeImage(imgData)
		if err != nil {
			log.Println("Error optimizing:", key, err)
			continue
		}

		dstKey := "optimized/" + key
		err = s3Client.UploadImage(ctx, dstKey, optimizedData)
		if err != nil {
			log.Println("Error uploading:", dstKey, err)
		} else {
			fmt.Println("Uploaded:", dstKey)
		}
	}
}

func LambdaHandler(ctx context.Context) error {
	srcBucket := os.Getenv("SOURCE_BUCKET")
	dstBucket := os.Getenv("DESTINATION_BUCKET")

	s3Client, err := NewS3Client(srcBucket, dstBucket)
	if err != nil {
		return fmt.Errorf("failed to initialize S3 client: %v", err)
	}

	imageKeys, err := s3Client.ListImages(ctx)
	if err != nil {
		return fmt.Errorf("failed to list images: %v", err)
	}
	if len(imageKeys) == 0 {
		fmt.Println("No images found. Exiting.")
		return nil
	}

	jobs := make(chan string, len(imageKeys))
	var wg sync.WaitGroup

	numWorkers := 3

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go Worker(ctx, s3Client, jobs, &wg)
	}

	for _, key := range imageKeys {
		jobs <- key
	}
	close(jobs) // close channel after sending all jobs

	wg.Wait() // wait for all workers

	fmt.Println("All images processed successfully!")
	return nil
}

func main() {
	lambda.Start(LambdaHandler)
}

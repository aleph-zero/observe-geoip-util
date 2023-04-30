package processors

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gammazero/workerpool"
	"log"
	"math/rand"
	"observe-geoip-util/options"
	"strings"
	"time"
)

const (
	randomSuffixLen  = 10
	fileNameTemplate = "networks-%s.ndjson"
)

func S3Processor(payload string, options options.Options, pool *workerpool.WorkerPool) {

	pool.Submit(func() {
		upload(payload, options)
	})
}

func upload(data string, options options.Options) {

	result, err := options.S3Uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(options.S3Bucket),
		Key:    aws.String(generateFilename()),
		Body:   strings.NewReader(data),
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("File uploaded to location: %s\n", result.Location)
}

func NewUploader(options options.Options) *s3manager.Uploader {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(options.AWSRegion),
		Credentials: credentials.NewStaticCredentials(
			options.AWSAccessKey,
			options.AWSSecretKey,
			""),
	})

	if err != nil {
		log.Fatal(err)
	}

	return s3manager.NewUploader(sess)
}

func generateFilename() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, randomSuffixLen)
	rand.Read(b)
	str := fmt.Sprintf("%x", b)[:randomSuffixLen]
	return fmt.Sprintf(fileNameTemplate, str)
}

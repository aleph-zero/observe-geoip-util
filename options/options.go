package options

import (
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Options struct {
	ObserveCustomerID  string
	ObserveIngestToken string
	S3Bucket           string
	AWSAccessKey       string
	AWSSecretKey       string
	AWSRegion          string
	S3Uploader         *s3manager.Uploader
	BatchSize          int
}

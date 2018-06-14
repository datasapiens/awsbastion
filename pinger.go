package awsbastion

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Pinger provides interface for structs that are able to verify given session
type Pinger interface {
	Ping(*session.Session) error
}

type S3ListObjectsPinger struct {
	Region, Bucket string
}

func (p *S3ListObjectsPinger) Ping(sess *session.Session) error {
	svc := s3.New(sess, &aws.Config{Region: aws.String(p.Region)})
	params := &s3.ListObjectsInput{
		Bucket: aws.String(p.Bucket),
	}
	_, err := svc.ListObjects(params)
	return err
}

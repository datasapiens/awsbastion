// Package awsbastion is solution for using golang AWS SDK for bastion account user's to assume main account role
// where everything is secured with MFA.
//
// It will prompt you for MFA device code on stdin and stores temporary credentials in file to be reused in next
// runs so user is not prompted all the time. This comes especially handy for local development.
//
// AWS Bastion
// A bastion account stores only IAM resources providing a central, isolated account. Users in the bastion account
// can access the resources in other accounts by assuming IAM roles into those accounts. These roles are setup to
// trust the bastion account to manage who is allowed to assume them and under what conditions they can be assumed,
// e.g. using temporary credentials with MFA.
// source: https://engineering.coinbase.com/you-need-more-than-one-aws-account-aws-bastions-and-assume-role-23946c6dfde3
package awsbastion

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

const filename = "bastion_credentials_session.json"

// Ping is function that verifies given session with aws servers e.g.
//
// func testConnection(region, bucket string) awsbastion.Ping {
//	return func(sess *session.Session) error {
//		svc := s3.New(sess, &aws.Config{Region: aws.String(region)})
//		params := &s3.ListObjectsInput{
//			Bucket: aws.String(bucket),
//		}
//		_, err := svc.ListObjects(params)
//		return err
//	}
//}
type Ping func(sess *session.Session) error

func storeCredentials(creds *credentials.Credentials) error {
	val, err := creds.Get()
	if err != nil {
		return fmt.Errorf("couldn't retrieve the credentials value: %v", err)
	}
	b, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("couldn't marshal credentials value: %v", err)
	}
	if err := ioutil.WriteFile(filename, b, 0666); err != nil {
		return fmt.Errorf("couldn't write the file: %v", err)
	}
	return nil
}

func retrieveCredentials() (*credentials.Credentials, error) {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("couldn't read file %s: %v", filename, err)
	}

	var val credentials.Value
	if err := json.Unmarshal(f, &val); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file: %v", err)
	}

	return credentials.NewCredentials(&credentials.StaticProvider{val}), nil
}

func bastionAccountCreds(profile, assumedRoleARN string) (*credentials.Credentials, error) {
	bastionCfg := aws.NewConfig()
	bastionOpts := session.Options{
		Config:                  *bastionCfg,
		Profile:                 profile,
		SharedConfigState:       session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}

	bastionSession, err := session.NewSessionWithOptions(bastionOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create new session for bastion account for profile %s: %v", profile, err)
	}

	creds := stscreds.NewCredentials(bastionSession, assumedRoleARN)
	if err := storeCredentials(creds); err != nil {
		return nil, fmt.Errorf("failed to store credentials for profile %s and role %s: %v", profile, assumedRoleARN, err)
	}
	return creds, nil
}

// Session creates session with empty config
// Create AWS session with given profile (as it is in .aws config) and assumed role's ARN from main account
func Session(profile, assumedRoleARN string, ping Ping) (*session.Session, error) {
	return SessionWithConfig(profile, assumedRoleARN, ping, aws.NewConfig())
}

// SessionWithConfig does same as Session but with custom config
func SessionWithConfig(profile, assumedRoleARN string, ping Ping, cfg *aws.Config) (*session.Session, error) {
	return sessionWithConfigWrapper(profile, assumedRoleARN, ping, cfg, false)
}

// SessionWithConfig does same as Session but with custom config
func sessionWithConfigWrapper(profile, assumedRoleARN string, ping Ping, cfg *aws.Config, rerun bool) (*session.Session, error) {
	creds, err := retrieveCredentials()
	if err != nil {
		creds, err = bastionAccountCreds(profile, assumedRoleARN)
		if err != nil {
			return nil, fmt.Errorf("couldn't create bastion account credentials: %v", err)
		}
	}

	mainSession, err := session.NewSession(cfg.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("couldn't create session: %v", err)
	}

	if err := ping(mainSession); err != nil {
		if rerun {
			return nil, fmt.Errorf("failed to ping aws servers with created session: %v")
		}
		if err := purge(); err != nil {
			return nil, fmt.Errorf("couldn't purge the main session: %v", err)
		}
		mainSession, err = sessionWithConfigWrapper(profile, assumedRoleARN, ping, cfg, true)
		if err != nil {
			return nil, fmt.Errorf("couldn't create main account session after first ping failed")
		}
	}

	return mainSession, nil
}

func purge() error {
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("couldn't delete file %s: %v", filename, err)
	}
	return nil
}

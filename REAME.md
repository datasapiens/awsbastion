# awsbastion
Package awsbastion is solution for using golang AWS SDK for bastion account user's to assume main account role
where everything is secured with MFA.

It will prompt you for MFA device code on stdin and stores temporary credentials in file to be reused in next
runs so user is not prompted all the time. This comes especially handy for local development.

### AWS Bastion
A bastion account stores only IAM resources providing a central, isolated account. Users in the bastion account
can access the resources in other accounts by assuming IAM roles into those accounts. These roles are setup to
trust the bastion account to manage who is allowed to assume them and under what conditions they can be assumed,
e.g. using temporary credentials with MFA.  
[source](https://engineering.coinbase.com/you-need-more-than-one-aws-account-aws-bastions-and-assume-role-23946c6dfde3)

Make sure you have bastion_credentials_session.json in `.gitignore`.

## Usage
```go
	cfg := &aws.Config{
		Region: aws.String(region),
	}
	roleARN := "arn:aws:iam::991941884292:role/power.assumerole"
	pinger := &awsbastion.S3ListObjectsPinger{region, bucket}
	sess, err := awsbastion.SessionWithConfig("poweruser", roleARN, pinger, cfg)
	if err != nil {
		panic(err)
	}
```

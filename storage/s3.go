package storage

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"google.golang.org/api/idtoken"
)

type GCPCredentialsProvider struct {
	ctx      context.Context
	RoleArn  string
	audience string
	region   string
}

func (p *GCPCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	// Testing to find the authorization type, need to be type = "service-account"
	// creds, _ := google.FindDefaultCredentials(ctx)
	// fmt.Println(string(creds.JSON))

	// rolesAnywhere method using Trust Anchor and using aws-signing-key
	// 	cfg, err := config.LoadDefaultConfig(ctx,
	// 		config.WithSharedConfigProfile("rolesanywhere"),
	// 	)
	// 	if err != nil {
	// 		log.Fatalf("failed to load AWS config: %v", err)
	// 	}

	ts, err := idtoken.NewTokenSource(ctx, p.audience)
	if err != nil {
		log.Fatalf("failed to create token source: %v", err)
	}

	tok, err := ts.Token()
	if err != nil {
		return aws.Credentials{}, err
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(p.region),
	)
	if err != nil {
		return aws.Credentials{}, err
	}

	stsClient := sts.NewFromConfig(cfg)

	out, err := stsClient.AssumeRoleWithWebIdentity(ctx, &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(p.RoleArn),
		RoleSessionName:  aws.String("gcp-session"),
		WebIdentityToken: aws.String(tok.AccessToken),
	})
	if err != nil {
		return aws.Credentials{}, err
	}

	// fmt.Println("Assumed role successfully:", *out.AssumedRoleUser.Arn)

	return aws.Credentials{
		AccessKeyID:     *out.Credentials.AccessKeyId,
		SecretAccessKey: *out.Credentials.SecretAccessKey,
		SessionToken:    *out.Credentials.SessionToken,
		CanExpire:       true,
		Expires:         *out.Credentials.Expiration,
	}, nil
}

func InitS3() *s3.Client {
	ctx := context.Background()

	region := os.Getenv("LOCATION")

	customProvider := &GCPCredentialsProvider{
		ctx:      ctx,
		RoleArn:  os.Getenv("AWS_ROLE_ARN"),
		audience: os.Getenv("AWS_OIDC_AUDIENCE"),
		region:   region,
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(customProvider)),
	)

	if err != nil {
		log.Fatal(err)
	}

	// s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
	// 	// Use the credentials from the AssumeRole output
	// 	o.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
	// 		*out.Credentials.AccessKeyId,
	// 		*out.Credentials.SecretAccessKey,
	// 		*out.Credentials.SessionToken,
	// 	))
	// })

	return s3.NewFromConfig(cfg)
}

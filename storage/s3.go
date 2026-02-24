package storage

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"google.golang.org/api/idtoken"
)

func InitS3() *s3.Client {
	ctx := context.Background()

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

	audience := os.Getenv("AWS_OIDC_AUDIENCE")
	roleArn := os.Getenv("AWS_ROLE_ARN")
	location := os.Getenv("LOCATION")

	ts, err := idtoken.NewTokenSource(ctx, audience)
	if err != nil {
		log.Fatalf("failed to create token source: %v", err)
	}

	tok, err := ts.Token()
	if err != nil {
		log.Fatalf("failed to get ID token: %v", err)
	}

	webToken := tok.AccessToken

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(location),
	)
	if err != nil {
		log.Fatal(err)
	}

	stsClient := sts.NewFromConfig(cfg)

	out, err := stsClient.AssumeRoleWithWebIdentity(ctx, &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(roleArn),
		RoleSessionName:  aws.String("gcp-session"),
		WebIdentityToken: aws.String(webToken),
	})
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Println("Assumed role successfully:", *out.AssumedRoleUser.Arn)

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        // Use the credentials from the AssumeRole output
        o.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
            *out.Credentials.AccessKeyId,
            *out.Credentials.SecretAccessKey,
            *out.Credentials.SessionToken,
        ))
    })

	// 	result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println("result = ", *result.Buckets[0].Name, *result.Buckets[1].Name)

    return s3Client
}

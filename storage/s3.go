package storage

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"google.golang.org/api/idtoken"
)

func InitS3() {
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

	fmt.Println("Assumed role successfully:", *out.AssumedRoleUser.Arn)

	s3Client := s3.NewFromConfig(cfg)

	result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)

	// bucket := "filedrive-bucket"
	// key := "test.txt"
	// content := "Hello from Roles Anywhere!"

	// _, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
	// 	Bucket: &bucket,
	// 	Key:    &key,
	// 	Body:   strings.NewReader(content),
	// })
	// if err != nil {
	// 	log.Fatalf("failed to upload: %v", err)
	// }

	// fmt.Println("File uploaded successfully!")
}

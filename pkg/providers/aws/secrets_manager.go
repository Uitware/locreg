package aws

import (
	"context"
	"encoding/json"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"log"
)

type SecretsManagerClient struct {
	client       *secretsmanager.Client
	locregConfig *parser.Config
}

type Secret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (secretM *SecretsManagerClient) createSecret(ctx context.Context, profile *parser.Profile) {
	secretData := Secret{
		Username: profile.LocalRegistry.Username,
		Password: profile.LocalRegistry.Password,
	}

	secretString, err := json.Marshal(secretData)
	if err != nil {
		log.Fatalf("Failed to marshal secret data")
	}

	resp, err := secretM.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:                        aws.String("LocregRegistrySecret" + parser.GenerateRandomString(5)),
		SecretString:                aws.String(string(secretString)),
		ForceOverwriteReplicaSecret: true,
		Tags:                        secretM.locregConfig.GenerateSecretTags(),
	})
	if err != nil {
		defer secretM.destroySecret(ctx, profile)
		log.Print("Failed to create secret: ", err)
		return
	}
	log.Printf("Secret ARN: %s", *resp.ARN)
	profile.AWSCloudResource.ECS.SecretARN = *resp.ARN
	log.Println(profile.AWSCloudResource.ECS.SecretARN)
	profile.Save()
	log.Println(profile.AWSCloudResource.ECS.SecretARN)

	log.Println("Secret created")
}

func (secretM *SecretsManagerClient) destroySecret(ctx context.Context, profile *parser.Profile) {
	retryOnError(5, 5, func() error {
		_, err := secretM.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			SecretId:             aws.String(profile.AWSCloudResource.ECS.SecretARN),
			RecoveryWindowInDays: aws.Int64(7),
		})
		return err
	})
	profile.AWSCloudResource.ECS.SecretARN = ""
	profile.Save()
}

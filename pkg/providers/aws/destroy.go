package aws

import (
	"context"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"log"
)

func Destroy(locregCfg *parser.Config) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	profile, _ := parser.LoadProfileData()
	if err != nil {
		log.Fatal("configuration error, " + err.Error())
	}

	ecsClient := ecs.NewFromConfig(cfg)
	ecsInstance := EcsClient{client: ecsClient}
	vpcInstance := VpcClient{client: ec2.NewFromConfig(cfg)}
	iamInstance := IamClient{client: iam.NewFromConfig(cfg), locregConfig: locregCfg}
	secretInstance := SecretsManagerClient{client: secretsmanager.NewFromConfig(cfg)}

	ecsInstance.destroyService(ctx, profile)
	ecsInstance.destroyTaskDefinition(ctx, profile)
	secretInstance.destroySecret(ctx, profile)
	iamInstance.destroyRole(ctx, profile)
	ecsInstance.destroyECS(ctx, profile)
	vpcInstance.destroyVpc(ctx, profile)
}

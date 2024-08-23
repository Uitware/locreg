package aws

import (
	"context"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"log"
)

func Destroy() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	profile, _ := parser.LoadProfileData()
	if err != nil {
		log.Fatal("configuration error, " + err.Error())
	}
	ecsClient := ecs.NewFromConfig(cfg)
	ecsInstance := EcsClient{client: ecsClient}
	ecsInstance.destroyService(ctx, profile)
	ecsInstance.destroyTaskDefinition(ctx, profile)
	ecsInstance.destroyECS(ctx, profile)
	vpcInstance := VpcClient{client: ec2.NewFromConfig(cfg)}
	vpcInstance.destroyVpc(ctx, profile)
}

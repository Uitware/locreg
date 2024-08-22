package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"log"
)

func Deploy() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal("configuration error, " + err.Error())
	}
	ecsClient := ecs.NewFromConfig(cfg)
	ecsInstance := EcsClient{client: ecsClient}
	subnetId := ecsInstance.deployECS(ctx, cfg)
	ecsInstance.runService(ctx, subnetId)
}

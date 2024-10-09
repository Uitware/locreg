package aws

import (
	"context"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"log"
)

func Deploy(locregCfg *parser.Config, envVars map[string]string) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(locregCfg.Deploy.Provider.AWS.Region))
	if err != nil {
		log.Fatal("configuration error, " + err.Error())
	}
	ecsClient := ecs.NewFromConfig(cfg)
	ecsInstance := EcsClient{
		client:       ecsClient,
		locregConfig: locregCfg,
	}
	subnetId := ecsInstance.deployECS(ctx, cfg, envVars)
	ecsInstance.runService(ctx, subnetId)
}

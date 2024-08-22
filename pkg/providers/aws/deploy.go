package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
	taskDetails := ecsInstance.runService(ctx, subnetId)
	vpcInstance := VpcClient{client: ec2.NewFromConfig(cfg)}
	containerIp := vpcInstance.getPublicIp(ctx, *taskDetails.Tasks[0].Attachments[0].Details[0].Value)
	log.Println("service running on ip: " + containerIp)
}

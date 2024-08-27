package aws

import (
	"context"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
	"strings"
	"time"
)

type VpcClient struct {
	client *ec2.Client
}

// createVpcForFargate creates a VPC in AWS and public subnet
// For containers that use the Fargate launch type.
//
// return: vpcId
func (vpcClient VpcClient) createVpcForFargate(ctx context.Context, profile *parser.Profile) *string {
	resp, err := vpcClient.client.CreateVpc(
		ctx,
		&ec2.CreateVpcInput{
			CidrBlock:         aws.String("10.10.0.0/16"),
			TagSpecifications: generateVPCTags(types.ResourceTypeVpc),
		})
	if err != nil {
		defer vpcClient.destroyVpc(ctx, profile)
		log.Printf("failed to create VPC, " + err.Error())
		return nil
	}
	profile.AWSCloudResource.VPCId = *resp.Vpc.VpcId
	profile.Save()

	// Get a default security group to configure it
	// inbound and outbound rules
	vpcSecurityGroup, err := vpcClient.client.DescribeSecurityGroups(
		ctx,
		&ec2.DescribeSecurityGroupsInput{
			Filters: []types.Filter{{
				Name:   aws.String("vpc-id"),
				Values: []string{*resp.Vpc.VpcId},
			}},
		})
	if err != nil {
		defer vpcClient.destroyVpc(ctx, profile)
		log.Println("failed to get security group, " + err.Error())
		return nil
	}

	_, err = vpcClient.client.AuthorizeSecurityGroupIngress(
		ctx,
		&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       vpcSecurityGroup.SecurityGroups[0].GroupId,
			IpPermissions: generateRulesForSG(),
		})
	if err != nil {
		defer vpcClient.destroyVpc(ctx, profile)
		log.Println("failed to authorize security group ingress, " + err.Error())
		return nil
	}

	_, err = vpcClient.client.AuthorizeSecurityGroupEgress(
		ctx,
		&ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       vpcSecurityGroup.SecurityGroups[0].GroupId,
			IpPermissions: generateRulesForSG(),
		})
	if err != nil {
		// Remove as not affecting the work of deployed containers
		// and don't affect anything
		if !strings.Contains(err.Error(), "InvalidPermission.Duplicate") {
			defer vpcClient.destroyVpc(ctx, profile)
			log.Println("failed to authorize security group egress, " + err.Error())
			return nil
		}
	}
	return resp.Vpc.VpcId
}

// createPublicSubnet creates a public subnet in the VPC
// For containers that use the Fargate launch type
func (vpcClient VpcClient) createPublicSubnet(ctx context.Context, profile *parser.Profile) string {
	vpcId := vpcClient.createVpcForFargate(ctx, profile)
	subnet, err := vpcClient.client.CreateSubnet(
		ctx,
		&ec2.CreateSubnetInput{
			VpcId:             vpcId,
			CidrBlock:         aws.String("10.10.10.0/24"),
			TagSpecifications: generateVPCTags(types.ResourceTypeSubnet),
		})
	if err != nil {
		defer vpcClient.deregisterAndDestroyFromVPC(ctx, profile)
		log.Println("failed to create subnet, " + err.Error())
		return ""
	}
	profile.AWSCloudResource.SubnetId = *subnet.Subnet.SubnetId
	profile.Save()

	internetGateway, err := vpcClient.client.CreateInternetGateway(
		ctx,
		&ec2.CreateInternetGatewayInput{
			TagSpecifications: generateVPCTags(types.ResourceTypeInternetGateway),
		})
	if err != nil {
		defer vpcClient.deregisterAndDestroyFromVPC(ctx, profile)
		log.Print("failed to create internet gateway, " + err.Error())
		return ""
	}
	profile.AWSCloudResource.InternetGatewayId = *internetGateway.InternetGateway.InternetGatewayId
	profile.Save()

	routeTable, err := vpcClient.client.CreateRouteTable(
		ctx,
		&ec2.CreateRouteTableInput{
			VpcId:             vpcId,
			TagSpecifications: generateVPCTags(types.ResourceTypeRouteTable),
		})
	if err != nil {
		defer vpcClient.deregisterAndDestroyFromVPC(ctx, profile)
		log.Println("failed to create route table, " + err.Error())
		return ""
	}
	profile.AWSCloudResource.RouteTableId = *routeTable.RouteTable.RouteTableId
	profile.Save()

	// First you need to attach the internet gateway to the VPC
	// only then you cat associate it with the route table
	_, err = vpcClient.client.AttachInternetGateway(
		ctx,
		&ec2.AttachInternetGatewayInput{
			VpcId:             vpcId,
			InternetGatewayId: internetGateway.InternetGateway.InternetGatewayId,
		})
	if err != nil {
		defer vpcClient.deregisterAndDestroyFromVPC(ctx, profile)
		log.Println("failed to attach internet gateway, " + err.Error())
		return ""
	}

	_, err = vpcClient.client.CreateRoute(
		ctx,
		&ec2.CreateRouteInput{
			RouteTableId:         routeTable.RouteTable.RouteTableId,
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
			GatewayId:            internetGateway.InternetGateway.InternetGatewayId,
		})
	if err != nil {
		defer vpcClient.deregisterAndDestroyFromVPC(ctx, profile)
		log.Println("failed to create route, " + err.Error())
		return ""
	}

	_, err = vpcClient.client.AssociateRouteTable(
		ctx,
		&ec2.AssociateRouteTableInput{
			RouteTableId: routeTable.RouteTable.RouteTableId,
			SubnetId:     subnet.Subnet.SubnetId,
		})
	if err != nil {
		defer vpcClient.deregisterAndDestroyFromVPC(ctx, profile)
		log.Println("failed to associate route table, " + err.Error())
		return ""
	}

	_, err = vpcClient.client.ModifySubnetAttribute(
		ctx,
		&ec2.ModifySubnetAttributeInput{
			SubnetId: subnet.Subnet.SubnetId,
			MapPublicIpOnLaunch: &types.AttributeBooleanValue{
				Value: aws.Bool(true),
			},
		})
	if err != nil {
		defer vpcClient.deregisterAndDestroyFromVPC(ctx, profile)
		log.Println("failed to modify subnet attribute, " + err.Error())
		return ""
	}
	log.Println("subnet created, " + *subnet.Subnet.SubnetId)
	return *subnet.Subnet.SubnetId
}

// deregisterAndDestroyFromVPC deregister and deletes Internet Gateway, RouteTable and Subnet from VPC
// that specified in the profile
func (vpcClient VpcClient) deregisterAndDestroyFromVPC(ctx context.Context, profile *parser.Profile) {
	// internetGateway
	retryOnError(5, 5, func() error {
		_, err := vpcClient.client.DetachInternetGateway(
			ctx,
			&ec2.DetachInternetGatewayInput{
				VpcId:             aws.String(profile.AWSCloudResource.VPCId),
				InternetGatewayId: aws.String(profile.AWSCloudResource.InternetGatewayId),
			})
		return err
	})

	_, err := vpcClient.client.DeleteInternetGateway(
		ctx,
		&ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(profile.AWSCloudResource.InternetGatewayId),
		})
	if err != nil {
		log.Fatal("failed to delete internet gateway, " + err.Error())
	}

	// Subnet must be deleted before route table because it is associated with
	// it and route table will not be deleted otherwise
	retryOnError(10, 5, func() error {
		_, err = vpcClient.client.DeleteSubnet(
			ctx,
			&ec2.DeleteSubnetInput{
				SubnetId: aws.String(profile.AWSCloudResource.SubnetId),
			})
		return err
	})

	// routeTable
	retryOnError(10, 5, func() error {
		_, err = vpcClient.client.DeleteRouteTable(
			ctx,
			&ec2.DeleteRouteTableInput{
				RouteTableId: aws.String(profile.AWSCloudResource.RouteTableId),
			})
		return err
	})
}

func (vpcClient VpcClient) destroyVpc(ctx context.Context, profile *parser.Profile) {
	vpcClient.deregisterAndDestroyFromVPC(ctx, profile)
	retryOnError(10, 5, func() error {
		_, err := vpcClient.client.DeleteVpc(
			ctx,
			&ec2.DeleteVpcInput{
				VpcId: aws.String(profile.AWSCloudResource.VPCId),
			})
		return err
	})
}

// generateVPCTags generates tags for the VPC and subnet and all other parts of networking that must be created
//
// TODO: make tags generate from one specified in config
func generateVPCTags(ragResourceType types.ResourceType) []types.TagSpecification {
	return []types.TagSpecification{
		{
			ResourceType: ragResourceType,
			Tags: []types.Tag{{
				Key:   aws.String("managed-by"),
				Value: aws.String("locreg")}},
		},
	}
}

// generateRulesForSG generates ingress and egress
// rules for the security group.
//
// TODO make it generate rules with values taken from config
func generateRulesForSG() []types.IpPermission {
	return []types.IpPermission{{
		FromPort:   aws.Int32(-1),
		ToPort:     aws.Int32(-1),
		IpProtocol: aws.String("-1"),
		IpRanges: []types.IpRange{{
			CidrIp:      aws.String("0.0.0.0/0"),
			Description: aws.String("allow all traffic"),
		}},
	}}
}

// retryOnError retries the function f if it returns an error
// retryTimes - number of times to retry
// sleepTime - time to sleep between retries
// f - function to retry
// used to retry on errors that are not critical usually for resource deletion
func retryOnError(retryTimes int, sleepTime int, f func() error) {
	for i := 0; i < retryTimes; i++ {
		err := f()
		if err != nil {
			//log.Println("failed to destroy resource, retrying... \n", err.Error()) // add debug mode for this
			log.Print("failed to destroy resource, retrying...")
			time.Sleep(time.Duration(i*sleepTime) * time.Second)
		} else {
			log.Println("resource destroyed successfully")
			break
		}
	}
}

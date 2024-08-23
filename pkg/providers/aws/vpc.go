package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
)

type VpcClient struct {
	client *ec2.Client
}

// createVpcForFargate creates a VPC in AWS and public subnet
// For containers that use the Fargate launch type.
//
// return: vpcId
func (vpcClient VpcClient) createVpcForFargate(ctx context.Context) *string {
	resp, err := vpcClient.client.CreateVpc(
		ctx,
		&ec2.CreateVpcInput{
			CidrBlock:         aws.String("10.10.0.0/16"),
			TagSpecifications: generateVPCTags(types.ResourceTypeVpc),
		})
	if err != nil {
		log.Fatal("failed to create VPC, " + err.Error())
	}

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
		log.Fatal("failed to get security group, " + err.Error())
	}

	_, err = vpcClient.client.AuthorizeSecurityGroupIngress(
		ctx,
		&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       vpcSecurityGroup.SecurityGroups[0].GroupId,
			IpPermissions: generateRulesForSG(),
		})
	if err != nil {
		log.Fatal("failed to authorize security group ingress, " + err.Error())
	}

	_, err = vpcClient.client.AuthorizeSecurityGroupEgress(
		ctx,
		&ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       vpcSecurityGroup.SecurityGroups[0].GroupId,
			IpPermissions: generateRulesForSG(),
		})
	if err != nil {
		log.Fatal("failed to authorize security group egress, " + err.Error())
	}
	return resp.Vpc.VpcId
}

// createPublicSubnet creates a public subnet in the VPC
// For containers that use the Fargate launch type
func (vpcClient VpcClient) createPublicSubnet(ctx context.Context) string {
	vpcId := vpcClient.createVpcForFargate(ctx)
	subnet, err := vpcClient.client.CreateSubnet(
		ctx,
		&ec2.CreateSubnetInput{
			VpcId:             vpcId,
			CidrBlock:         aws.String("10.10.10.0/24"),
			TagSpecifications: generateVPCTags(types.ResourceTypeSubnet),
		})
	if err != nil {
		log.Fatal("failed to create subnet, " + err.Error())
	}
	internetGateway, err := vpcClient.client.CreateInternetGateway(
		ctx,
		&ec2.CreateInternetGatewayInput{
			TagSpecifications: generateVPCTags(types.ResourceTypeInternetGateway),
		})
	if err != nil {
		log.Fatal("failed to create internet gateway, " + err.Error())
	}

	routeTable, err := vpcClient.client.CreateRouteTable(
		ctx,
		&ec2.CreateRouteTableInput{
			VpcId:             vpcId,
			TagSpecifications: generateVPCTags(types.ResourceTypeRouteTable),
		})
	if err != nil {
		log.Fatal("failed to create route table, " + err.Error())
	}

	// First you need to attach the internet gateway to the VPC
	// only then you cat associate it with the route table
	_, err = vpcClient.client.AttachInternetGateway(
		ctx,
		&ec2.AttachInternetGatewayInput{
			VpcId:             vpcId,
			InternetGatewayId: internetGateway.InternetGateway.InternetGatewayId,
		})
	if err != nil {
		log.Fatal("failed to attach internet gateway, " + err.Error())
	}

	_, err = vpcClient.client.CreateRoute(
		ctx,
		&ec2.CreateRouteInput{
			RouteTableId:         routeTable.RouteTable.RouteTableId,
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
			GatewayId:            internetGateway.InternetGateway.InternetGatewayId,
		})
	if err != nil {
		log.Fatal("failed to create route, " + err.Error())
	}

	_, err = vpcClient.client.AssociateRouteTable(
		ctx,
		&ec2.AssociateRouteTableInput{
			RouteTableId: routeTable.RouteTable.RouteTableId,
			SubnetId:     subnet.Subnet.SubnetId,
		})
	if err != nil {
		log.Fatal("failed to associate route table, " + err.Error())
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
		log.Fatal("failed to modify subnet attribute, " + err.Error())
	}
	log.Println("subnet created, " + *subnet.Subnet.SubnetId)
	return *subnet.Subnet.SubnetId
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

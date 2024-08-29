package parser

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// Methods related to AWS configuration for the parser

// GenerateContainerPorts generates the container ports mappings for the task definition
func (config *Config) GenerateContainerPorts() []types.PortMapping {
	portMappings := make([]types.PortMapping, 0, len(config.Deploy.Provider.AWS.ECS.TaskDefinition.ContainerDefinition.PortMappings))

	for _, st := range config.Deploy.Provider.AWS.ECS.TaskDefinition.ContainerDefinition.PortMappings {
		switch st.Protocol {
		case "tcp":
			portMappings = append(portMappings, types.PortMapping{
				ContainerPort: aws.Int32(int32(st.ContainerPort)),
				HostPort:      aws.Int32(int32(st.HostPort)),
				Protocol:      types.TransportProtocolTcp,
			})
		case "udp":
			portMappings = append(portMappings, types.PortMapping{
				ContainerPort: aws.Int32(int32(st.ContainerPort)),
				HostPort:      aws.Int32(int32(st.HostPort)),
				Protocol:      types.TransportProtocolUdp,
			})
		}
	}
	return portMappings
}

// GenerateECSTags generates tags for the ECS cluster and task definition that is created
func (config *Config) GenerateECSTags() []types.Tag {
	tags := make([]types.Tag, 0, len(config.Tags))

	for key, value := range config.Tags {
		tags = append(tags, types.Tag{
			Key:   aws.String(key),
			Value: value,
		})
	}

	return tags
}

// GenerateVPCTags generates tags for the VPC and subnet and all other parts of networking that must be created
func (config *Config) GenerateVPCTags(tagResourceType ec2Types.ResourceType) []ec2Types.TagSpecification {
	tags := make([]ec2Types.Tag, 0, len(config.Tags))

	for key, value := range config.Tags {
		tags = append(tags, ec2Types.Tag{
			Key:   aws.String(key),
			Value: value,
		})
	}

	return []ec2Types.TagSpecification{
		{
			Tags:         tags,
			ResourceType: tagResourceType,
		},
	}
}

// GenerateRulesForSG generates ingress and egress rules for a default security group.
// Generate rules to allow traffic only to ports that are exposed on container
func (config *Config) GenerateRulesForSG() []ec2Types.IpPermission {
	rules := make([]ec2Types.IpPermission, 0, len(config.Deploy.Provider.AWS.ECS.TaskDefinition.ContainerDefinition.PortMappings))

	for _, port := range config.Deploy.Provider.AWS.ECS.TaskDefinition.ContainerDefinition.PortMappings {
		rules = append(rules, ec2Types.IpPermission{
			FromPort:   aws.Int32(int32(port.HostPort)),
			ToPort:     aws.Int32(int32(port.HostPort)),
			IpProtocol: aws.String(port.Protocol),
			IpRanges: []ec2Types.IpRange{{
				CidrIp:      aws.String("0.0.0.0/0"),
				Description: aws.String("allow traffic from all IPs for specified port"),
			}},
		})
	}

	return rules
}

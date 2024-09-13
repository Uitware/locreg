package aws

import (
	"context"
	"encoding/json"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"log"
)

type IamClient struct {
	client       *iam.Client
	locregConfig *parser.Config
}

type PolicyDocument struct {
	Version   string
	Statement []PolicyStatement
}

// PolicyStatement defines a statement in a policy document.
type PolicyStatement struct {
	Effect    string
	Action    []string
	Principal map[string]string `json:",omitempty"`
	Resource  *string           `json:",omitempty"`
}

func (iamClient IamClient) createRole(ctx context.Context, profile *parser.Profile) {
	trustPolicy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{{
			Effect:    "Allow",
			Principal: map[string]string{"Service": "ecs-tasks.amazonaws.com"},
			Action:    []string{"sts:AssumeRole"},
		}},
	}

	policyBytes, err := json.Marshal(trustPolicy)
	if err != nil {
		log.Print("Failed to marshal trust policy: ", err)
		return
	}

	role, err := iamClient.client.CreateRole(ctx, &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(policyBytes)),
		RoleName:                 aws.String(iamClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.IAMRoleName),
	})
	if err != nil {
		defer Destroy(iamClient.locregConfig)
		log.Print("Failed to create role: ", err)
		return
	}
	log.Println(*role.Role.Arn)
	profile.AWSCloudResource.ECS.RoleARN = *role.Role.Arn
	profile.Save()

	secretManagerPolicy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{{
			Effect:   "Allow",
			Action:   []string{"secretsmanager:GetSecretValue"},
			Resource: aws.String("*"),
		}},
	}

	policyDocBytes, err := json.Marshal(secretManagerPolicy)
	if err != nil {
		defer iamClient.destroyRole(ctx, profile)
		log.Print("Failed to marshal secrets manager policy: ", err)
		return
	}

	_, err = iamClient.client.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       role.Role.RoleName,
		PolicyName:     aws.String("SecretsManagerAccessPolicy"),
		PolicyDocument: aws.String(string(policyDocBytes)),
	})
	if err != nil {
		defer Destroy(iamClient.locregConfig)
		log.Print("Failed to attach policy to role: ", err)
		return
	}
	log.Println("Successfully created IAM role and attached policy.")
}

func (iamClient IamClient) destroyRole(ctx context.Context, profile *parser.Profile) {
	// Delete inline policy "SecretsManagerAccessPolicy"
	retryOnError(5, 5, func() error {
		_, err := iamClient.client.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
			PolicyName: aws.String("SecretsManagerAccessPolicy"),
			RoleName:   aws.String(iamClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.IAMRoleName),
		})
		if err != nil {
			log.Print("Failed to delete inline policy: ", err)
		}
		return err
	})

	// List and detach managed policies from the role
	retryOnError(5, 5, func() error {
		listPolicies, err := iamClient.client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(iamClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.IAMRoleName),
		})
		if err != nil {
			log.Print("Failed to list attached policies: ", err)
			return err
		}

		for _, policy := range listPolicies.AttachedPolicies {
			_, err = iamClient.client.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				RoleName:  aws.String(iamClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.IAMRoleName),
				PolicyArn: policy.PolicyArn,
			})
			if err != nil {
				log.Print("Failed to detach policy: ", err)
				return err
			}
		}
		return nil
	})

	// Delete the role itself
	retryOnError(5, 5, func() error {
		_, err := iamClient.client.DeleteRole(ctx, &iam.DeleteRoleInput{
			RoleName: aws.String(iamClient.locregConfig.Deploy.Provider.AWS.ECS.TaskDefinition.IAMRoleName),
		})
		if err != nil {
			log.Print("Failed to delete role: ", err)
		}
		return err
	})

	// Clear the RoleARN and save profile
	profile.AWSCloudResource.ECS.RoleARN = ""
	profile.Save()
	log.Println("Successfully deleted IAM role and updated profile.")
}

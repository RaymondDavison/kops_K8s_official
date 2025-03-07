/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package awsup

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/aws/aws-sdk-go/service/eventbridge/eventbridgeiface"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kops/dnsprovider/pkg/dnsprovider"
	dnsproviderroute53 "k8s.io/kops/dnsprovider/pkg/dnsprovider/providers/aws/route53"
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/pkg/cloudinstances"
	"k8s.io/kops/pkg/resources/spotinst"
	"k8s.io/kops/upup/pkg/fi"
)

type MockAWSCloud struct {
	MockCloud
	region string
	tags   map[string]string

	zones []*ec2.AvailabilityZone
}

var _ fi.Cloud = (*MockAWSCloud)(nil)

func InstallMockAWSCloud(region string, zoneLetters string) *MockAWSCloud {
	i := BuildMockAWSCloud(region, zoneLetters)
	awsCloudInstances[region] = i
	allRegions = []*ec2.Region{
		{RegionName: aws.String(region)},
	}
	return i
}

func BuildMockAWSCloud(region string, zoneLetters string) *MockAWSCloud {
	i := &MockAWSCloud{region: region}
	for _, c := range zoneLetters {
		azName := fmt.Sprintf("%s%c", region, c)
		az := &ec2.AvailabilityZone{
			RegionName: aws.String(region),
			ZoneName:   aws.String(azName),
			State:      aws.String("available"),
		}
		i.zones = append(i.zones, az)
	}
	return i
}

type MockCloud struct {
	MockAutoscaling    autoscalingiface.AutoScalingAPI
	MockCloudFormation *cloudformation.CloudFormation
	MockEC2            ec2iface.EC2API
	MockIAM            iamiface.IAMAPI
	MockRoute53        route53iface.Route53API
	MockELB            elbiface.ELBAPI
	MockELBV2          elbv2iface.ELBV2API
	MockSpotinst       spotinst.Cloud
	MockSQS            sqsiface.SQSAPI
	MockEventBridge    eventbridgeiface.EventBridgeAPI
}

func (c *MockAWSCloud) DeleteGroup(g *cloudinstances.CloudInstanceGroup) error {
	return deleteGroup(c, g)
}

func (c *MockAWSCloud) DeleteInstance(i *cloudinstances.CloudInstance) error {
	return deleteInstance(c, i)
}

func (c *MockAWSCloud) DeregisterInstance(i *cloudinstances.CloudInstance) error {
	return nil
}

func (c *MockAWSCloud) DetachInstance(i *cloudinstances.CloudInstance) error {
	return detachInstance(c, i)
}

func (c *MockAWSCloud) GetCloudGroups(cluster *kops.Cluster, instancegroups []*kops.InstanceGroup, warnUnmatched bool, nodes []v1.Node) (map[string]*cloudinstances.CloudInstanceGroup, error) {
	return getCloudGroups(c, cluster, instancegroups, warnUnmatched, nodes)
}

func (c *MockCloud) ProviderID() kops.CloudProviderID {
	return kops.CloudProviderAWS
}

func (c *MockCloud) DNS() (dnsprovider.Interface, error) {
	if c.MockRoute53 == nil {
		return nil, fmt.Errorf("MockRoute53 not set")
	}
	return dnsproviderroute53.New(c.MockRoute53), nil
}

func (c *MockAWSCloud) Region() string {
	return c.region
}

func (c *MockAWSCloud) DescribeAvailabilityZones() ([]*ec2.AvailabilityZone, error) {
	return c.zones, nil
}

func (c *MockAWSCloud) AddTags(name *string, tags map[string]string) {
	if name != nil {
		tags["Name"] = *name
	}
	for k, v := range c.tags {
		tags[k] = v
	}
}

func (c *MockAWSCloud) BuildFilters(name *string) []*ec2.Filter {
	return buildFilters(c.tags, name)
}

func (c *MockAWSCloud) AddAWSTags(id string, expected map[string]string) error {
	return addAWSTags(c, id, expected)
}

func (c *MockAWSCloud) DeleteTags(id string, tags map[string]string) error {
	return deleteTags(c, id, tags)
}

func (c *MockAWSCloud) BuildTags(name *string) map[string]string {
	return buildTags(c.tags, name)
}

func (c *MockAWSCloud) Tags() map[string]string {
	tags := make(map[string]string)
	for k, v := range c.tags {
		tags[k] = v
	}
	return tags
}

func (c *MockAWSCloud) CreateTags(resourceId string, tags map[string]string) error {
	return createTags(c, resourceId, tags)
}

func (c *MockAWSCloud) GetTags(resourceID string) (map[string]string, error) {
	return getTags(c, resourceID)
}

func (c *MockAWSCloud) UpdateTags(id string, tags map[string]string) error {
	return updateTags(c, id, tags)
}

func (c *MockAWSCloud) GetELBTags(loadBalancerName string) (map[string]string, error) {
	return getELBTags(c, loadBalancerName)
}

func (c *MockAWSCloud) CreateELBTags(loadBalancerName string, tags map[string]string) error {
	return createELBTags(c, loadBalancerName, tags)
}

func (c *MockAWSCloud) RemoveELBTags(loadBalancerName string, tags map[string]string) error {
	return removeELBTags(c, loadBalancerName, tags)
}

func (c *MockAWSCloud) GetELBV2Tags(ResourceArn string) (map[string]string, error) {
	return getELBV2Tags(c, ResourceArn)
}

func (c *MockAWSCloud) CreateELBV2Tags(ResourceArn string, tags map[string]string) error {
	return createELBV2Tags(c, ResourceArn, tags)
}

func (c *MockAWSCloud) RemoveELBV2Tags(ResourceArn string, tags map[string]string) error {
	return removeELBV2Tags(c, ResourceArn, tags)
}

func (c *MockAWSCloud) FindELBByNameTag(findNameTag string) (*elb.LoadBalancerDescription, error) {
	return findELBByNameTag(c, findNameTag)
}

func (c *MockAWSCloud) DescribeELBTags(loadBalancerNames []string) (map[string][]*elb.Tag, error) {
	return describeELBTags(c, loadBalancerNames)
}

func (c *MockAWSCloud) FindELBV2ByNameTag(findNameTag string) (*elbv2.LoadBalancer, error) {
	return findELBV2ByNameTag(c, findNameTag)
}

func (c *MockAWSCloud) DescribeELBV2Tags(loadBalancerArns []string) (map[string][]*elbv2.Tag, error) {
	return describeELBV2Tags(c, loadBalancerArns)
}

func (c *MockAWSCloud) DescribeInstance(instanceID string) (*ec2.Instance, error) {
	return nil, fmt.Errorf("MockAWSCloud DescribeInstance not implemented")
}

func (c *MockAWSCloud) DescribeVPC(vpcID string) (*ec2.Vpc, error) {
	return describeVPC(c, vpcID)
}

func (c *MockAWSCloud) ResolveImage(name string) (*ec2.Image, error) {
	return resolveImage(c.MockEC2, name)
}

func (c *MockAWSCloud) WithTags(tags map[string]string) AWSCloud {
	m := &MockAWSCloud{}
	*m = *c
	m.tags = tags
	return m
}

func (c *MockAWSCloud) CloudFormation() *cloudformation.CloudFormation {
	if c.MockEC2 == nil {
		klog.Fatalf("MockAWSCloud MockCloudFormation not set")
	}
	return c.MockCloudFormation
}

func (c *MockAWSCloud) EC2() ec2iface.EC2API {
	if c.MockEC2 == nil {
		klog.Fatalf("MockAWSCloud MockEC2 not set")
	}
	return c.MockEC2
}

func (c *MockAWSCloud) IAM() iamiface.IAMAPI {
	if c.MockIAM == nil {
		klog.Fatalf("MockAWSCloud MockIAM not set")
	}
	return c.MockIAM
}

func (c *MockAWSCloud) ELB() elbiface.ELBAPI {
	if c.MockELB == nil {
		klog.Fatalf("MockAWSCloud MockELB not set")
	}
	return c.MockELB
}

func (c *MockAWSCloud) ELBV2() elbv2iface.ELBV2API {
	if c.MockELBV2 == nil {
		klog.Fatalf("MockAWSCloud MockELBV2 not set")
	}
	return c.MockELBV2
}

func (c *MockAWSCloud) Autoscaling() autoscalingiface.AutoScalingAPI {
	if c.MockAutoscaling == nil {
		klog.Fatalf("MockAWSCloud Autoscaling not set")
	}
	return c.MockAutoscaling
}

func (c *MockAWSCloud) Route53() route53iface.Route53API {
	if c.MockRoute53 == nil {
		klog.Fatalf("MockRoute53 not set")
	}
	return c.MockRoute53
}

func (c *MockAWSCloud) Spotinst() spotinst.Cloud {
	if c.MockSpotinst == nil {
		klog.Fatalf("MockSpotinst not set")
	}
	return c.MockSpotinst
}

func (c *MockAWSCloud) SQS() sqsiface.SQSAPI {
	if c.MockSQS == nil {
		klog.Fatalf("MockSQS not set")
	}
	return c.MockSQS
}

func (c *MockAWSCloud) EventBridge() eventbridgeiface.EventBridgeAPI {
	if c.MockEventBridge == nil {
		klog.Fatalf("MockEventBridgess not set")
	}
	return c.MockEventBridge
}

func (c *MockAWSCloud) FindVPCInfo(id string) (*fi.VPCInfo, error) {
	return findVPCInfo(c, id)
}

func (c *MockAWSCloud) GetApiIngressStatus(cluster *kops.Cluster) ([]fi.ApiIngressStatus, error) {
	return getApiIngressStatus(c, cluster)
}

// DefaultInstanceType determines an instance type for the specified cluster & instance group
func (c *MockAWSCloud) DefaultInstanceType(cluster *kops.Cluster, ig *kops.InstanceGroup) (string, error) {
	switch ig.Spec.Role {
	case kops.InstanceGroupRoleMaster:
		return "m3.medium", nil
	case kops.InstanceGroupRoleNode:
		return "t2.medium", nil
	case kops.InstanceGroupRoleBastion:
		return "t2.micro", nil
	default:
		return "", fmt.Errorf("MockAWSCloud DefaultInstanceType does not handle %s", ig.Spec.Role)
	}
}

// DescribeInstanceType calls ec2.DescribeInstanceType to get information for a particular instance type
func (c *MockAWSCloud) DescribeInstanceType(instanceType string) (*ec2.InstanceTypeInfo, error) {
	if instanceType == "t2.invalidType" {
		return nil, fmt.Errorf("invalid instance type specified: t2.invalidType")
	}
	info := &ec2.InstanceTypeInfo{
		NetworkInfo: &ec2.NetworkInfo{
			MaximumNetworkInterfaces:  aws.Int64(1),
			Ipv4AddressesPerInterface: aws.Int64(1),
		},
		MemoryInfo: &ec2.MemoryInfo{
			SizeInMiB: aws.Int64(1024),
		},
		VCpuInfo: &ec2.VCpuInfo{
			DefaultVCpus: aws.Int64(2),
		},
	}
	if instanceType == "m3.medium" {
		info.InstanceStorageInfo = &ec2.InstanceStorageInfo{
			Disks: []*ec2.DiskInfo{
				{
					Count:    aws.Int64(1),
					SizeInGB: aws.Int64(1024),
				},
			},
		}
	}

	switch instanceType {
	case "c5.large", "m3.medium", "m4.large", "m5.large", "m5.xlarge", "t3.micro", "t3.medium", "t3.large", "c4.large":
		info.ProcessorInfo = &ec2.ProcessorInfo{
			SupportedArchitectures: []*string{
				aws.String(ec2.ArchitectureTypeX8664),
			},
		}
	case "a1.large":
		info.ProcessorInfo = &ec2.ProcessorInfo{
			SupportedArchitectures: []*string{
				aws.String(ec2.ArchitectureTypeArm64),
			},
		}
	case "t2.micro", "t2.medium":
		info.ProcessorInfo = &ec2.ProcessorInfo{
			SupportedArchitectures: []*string{
				aws.String(ec2.ArchitectureTypeI386),
				aws.String(ec2.ArchitectureTypeX8664),
			},
		}
	case "g4dn.xlarge", "g4ad.16xlarge":
		info.ProcessorInfo = &ec2.ProcessorInfo{
			SupportedArchitectures: []*string{
				aws.String(ec2.ArchitectureTypeX8664),
			},
		}
		info.GpuInfo = &ec2.GpuInfo{}
	}

	return info, nil
}

// AccountInfo returns the AWS account ID and AWS partition that we are deploying into
func (c *MockAWSCloud) AccountInfo() (string, string, error) {
	return "123456789012", "aws-test", nil
}

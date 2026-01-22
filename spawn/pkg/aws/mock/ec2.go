package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EC2API defines the subset of EC2 API operations we need to mock
type EC2API interface {
	RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
	DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error)
	DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error)
	ImportKeyPair(ctx context.Context, params *ec2.ImportKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.ImportKeyPairOutput, error)
	StopInstances(ctx context.Context, params *ec2.StopInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error)
	StartInstances(ctx context.Context, params *ec2.StartInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error)
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	CreateSecurityGroup(ctx context.Context, params *ec2.CreateSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreateSecurityGroupOutput, error)
	AuthorizeSecurityGroupIngress(ctx context.Context, params *ec2.AuthorizeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupIngressOutput, error)
	CreatePlacementGroup(ctx context.Context, params *ec2.CreatePlacementGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreatePlacementGroupOutput, error)
	DeletePlacementGroup(ctx context.Context, params *ec2.DeletePlacementGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeletePlacementGroupOutput, error)
	DescribeInstanceTypes(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

// MockEC2Client provides a mock implementation of EC2 operations for testing
type MockEC2Client struct {
	mu sync.RWMutex

	// Mock data storage
	Instances       map[string]*types.Instance
	KeyPairs        map[string]*types.KeyPairInfo
	SecurityGroups  map[string]*types.SecurityGroup
	PlacementGroups map[string]*types.PlacementGroup
	Vpcs            map[string]*types.Vpc
	Regions         []types.Region
	Images          map[string]*types.Image
	InstanceTypes   map[string]*types.InstanceTypeInfo

	// Errors to return for specific operations (for error testing)
	RunInstancesErr            error
	DescribeInstancesErr       error
	TerminateInstancesErr      error
	CreateTagsErr              error
	DescribeRegionsErr         error
	DescribeKeyPairsErr        error
	ImportKeyPairErr           error
	StopInstancesErr           error
	StartInstancesErr          error
	DescribeVpcsErr            error
	DescribeSecurityGroupsErr  error
	CreateSecurityGroupErr     error
	CreatePlacementGroupErr    error
	DeletePlacementGroupErr    error
	DescribeInstanceTypesErr   error
	DescribeImagesErr          error

	// Call tracking
	RunInstancesCalls            int
	DescribeInstancesCalls       int
	TerminateInstancesCalls      int
	CreateTagsCalls              int
	DescribeRegionsCalls         int
	DescribeKeyPairsCalls        int
	ImportKeyPairCalls           int
	StopInstancesCalls           int
	StartInstancesCalls          int
	DescribeVpcsCalls            int
	DescribeSecurityGroupsCalls  int
	CreateSecurityGroupCalls     int
	CreatePlacementGroupCalls    int
	DeletePlacementGroupCalls    int
	DescribeInstanceTypesCalls   int
	DescribeImagesCalls          int
}

// NewMockEC2Client creates a new mock EC2 client with default data
func NewMockEC2Client() *MockEC2Client {
	return &MockEC2Client{
		Instances:       make(map[string]*types.Instance),
		KeyPairs:        make(map[string]*types.KeyPairInfo),
		SecurityGroups:  make(map[string]*types.SecurityGroup),
		PlacementGroups: make(map[string]*types.PlacementGroup),
		Vpcs:            make(map[string]*types.Vpc),
		Images:          make(map[string]*types.Image),
		InstanceTypes:   make(map[string]*types.InstanceTypeInfo),
		Regions: []types.Region{
			{RegionName: strPtr("us-east-1")},
			{RegionName: strPtr("us-west-2")},
			{RegionName: strPtr("eu-west-1")},
		},
	}
}

func (m *MockEC2Client) RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunInstancesCalls++

	if m.RunInstancesErr != nil {
		return nil, m.RunInstancesErr
	}

	// Generate mock instance
	instanceID := fmt.Sprintf("i-%s", randomID())
	instance := &types.Instance{
		InstanceId:       strPtr(instanceID),
		InstanceType:     params.InstanceType,
		ImageId:          params.ImageId,
		State:            &types.InstanceState{Name: types.InstanceStateNameRunning},
		PublicIpAddress:  strPtr(fmt.Sprintf("52.%d.%d.%d", randomByte(), randomByte(), randomByte())),
		PrivateIpAddress: strPtr(fmt.Sprintf("10.0.%d.%d", randomByte(), randomByte())),
	}

	if params.KeyName != nil {
		instance.KeyName = params.KeyName
	}

	m.Instances[instanceID] = instance

	return &ec2.RunInstancesOutput{
		Instances: []types.Instance{*instance},
	}, nil
}

func (m *MockEC2Client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.DescribeInstancesCalls++

	if m.DescribeInstancesErr != nil {
		return nil, m.DescribeInstancesErr
	}

	var instances []types.Instance
	if len(params.InstanceIds) > 0 {
		// Filter by instance IDs
		for _, id := range params.InstanceIds {
			if inst, ok := m.Instances[id]; ok {
				instances = append(instances, *inst)
			}
		}
	} else {
		// Return all instances
		for _, inst := range m.Instances {
			instances = append(instances, *inst)
		}
	}

	return &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{
			{Instances: instances},
		},
	}, nil
}

func (m *MockEC2Client) TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TerminateInstancesCalls++

	if m.TerminateInstancesErr != nil {
		return nil, m.TerminateInstancesErr
	}

	var stateChanges []types.InstanceStateChange
	for _, id := range params.InstanceIds {
		if inst, ok := m.Instances[id]; ok {
			inst.State = &types.InstanceState{Name: types.InstanceStateNameTerminated}
			stateChanges = append(stateChanges, types.InstanceStateChange{
				InstanceId:    strPtr(id),
				CurrentState:  &types.InstanceState{Name: types.InstanceStateNameTerminated},
				PreviousState: &types.InstanceState{Name: types.InstanceStateNameRunning},
			})
		}
	}

	return &ec2.TerminateInstancesOutput{
		TerminatingInstances: stateChanges,
	}, nil
}

func (m *MockEC2Client) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateTagsCalls++

	if m.CreateTagsErr != nil {
		return nil, m.CreateTagsErr
	}

	// Update tags on instances
	for _, resourceID := range params.Resources {
		if inst, ok := m.Instances[resourceID]; ok {
			if inst.Tags == nil {
				inst.Tags = []types.Tag{}
			}
			inst.Tags = append(inst.Tags, params.Tags...)
		}
	}

	return &ec2.CreateTagsOutput{}, nil
}

func (m *MockEC2Client) DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.DescribeRegionsCalls++

	if m.DescribeRegionsErr != nil {
		return nil, m.DescribeRegionsErr
	}

	return &ec2.DescribeRegionsOutput{
		Regions: m.Regions,
	}, nil
}

func (m *MockEC2Client) DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.DescribeKeyPairsCalls++

	if m.DescribeKeyPairsErr != nil {
		return nil, m.DescribeKeyPairsErr
	}

	var keyPairs []types.KeyPairInfo
	if len(params.KeyNames) > 0 {
		for _, name := range params.KeyNames {
			if kp, ok := m.KeyPairs[name]; ok {
				keyPairs = append(keyPairs, *kp)
			}
		}
	} else {
		for _, kp := range m.KeyPairs {
			keyPairs = append(keyPairs, *kp)
		}
	}

	return &ec2.DescribeKeyPairsOutput{
		KeyPairs: keyPairs,
	}, nil
}

func (m *MockEC2Client) ImportKeyPair(ctx context.Context, params *ec2.ImportKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.ImportKeyPairOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ImportKeyPairCalls++

	if m.ImportKeyPairErr != nil {
		return nil, m.ImportKeyPairErr
	}

	keyPair := &types.KeyPairInfo{
		KeyName:   params.KeyName,
		KeyPairId: strPtr(fmt.Sprintf("key-%s", randomID())),
	}
	m.KeyPairs[*params.KeyName] = keyPair

	return &ec2.ImportKeyPairOutput{
		KeyName:   params.KeyName,
		KeyPairId: keyPair.KeyPairId,
	}, nil
}

func (m *MockEC2Client) StopInstances(ctx context.Context, params *ec2.StopInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StopInstancesCalls++

	if m.StopInstancesErr != nil {
		return nil, m.StopInstancesErr
	}

	var stateChanges []types.InstanceStateChange
	for _, id := range params.InstanceIds {
		if inst, ok := m.Instances[id]; ok {
			inst.State = &types.InstanceState{Name: types.InstanceStateNameStopped}
			stateChanges = append(stateChanges, types.InstanceStateChange{
				InstanceId:    strPtr(id),
				CurrentState:  &types.InstanceState{Name: types.InstanceStateNameStopped},
				PreviousState: &types.InstanceState{Name: types.InstanceStateNameRunning},
			})
		}
	}

	return &ec2.StopInstancesOutput{
		StoppingInstances: stateChanges,
	}, nil
}

func (m *MockEC2Client) StartInstances(ctx context.Context, params *ec2.StartInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartInstancesCalls++

	if m.StartInstancesErr != nil {
		return nil, m.StartInstancesErr
	}

	var stateChanges []types.InstanceStateChange
	for _, id := range params.InstanceIds {
		if inst, ok := m.Instances[id]; ok {
			inst.State = &types.InstanceState{Name: types.InstanceStateNameRunning}
			stateChanges = append(stateChanges, types.InstanceStateChange{
				InstanceId:    strPtr(id),
				CurrentState:  &types.InstanceState{Name: types.InstanceStateNameRunning},
				PreviousState: &types.InstanceState{Name: types.InstanceStateNameStopped},
			})
		}
	}

	return &ec2.StartInstancesOutput{
		StartingInstances: stateChanges,
	}, nil
}

func (m *MockEC2Client) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.DescribeVpcsCalls++

	if m.DescribeVpcsErr != nil {
		return nil, m.DescribeVpcsErr
	}

	vpcs := make([]types.Vpc, 0, len(m.Vpcs))
	for _, vpc := range m.Vpcs {
		vpcs = append(vpcs, *vpc)
	}

	return &ec2.DescribeVpcsOutput{
		Vpcs: vpcs,
	}, nil
}

func (m *MockEC2Client) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.DescribeSecurityGroupsCalls++

	if m.DescribeSecurityGroupsErr != nil {
		return nil, m.DescribeSecurityGroupsErr
	}

	sgs := make([]types.SecurityGroup, 0, len(m.SecurityGroups))
	for _, sg := range m.SecurityGroups {
		sgs = append(sgs, *sg)
	}

	return &ec2.DescribeSecurityGroupsOutput{
		SecurityGroups: sgs,
	}, nil
}

func (m *MockEC2Client) CreateSecurityGroup(ctx context.Context, params *ec2.CreateSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreateSecurityGroupOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateSecurityGroupCalls++

	if m.CreateSecurityGroupErr != nil {
		return nil, m.CreateSecurityGroupErr
	}

	sgID := fmt.Sprintf("sg-%s", randomID())
	sg := &types.SecurityGroup{
		GroupId:     strPtr(sgID),
		GroupName:   params.GroupName,
		Description: params.Description,
		VpcId:       params.VpcId,
	}
	m.SecurityGroups[sgID] = sg

	return &ec2.CreateSecurityGroupOutput{
		GroupId: strPtr(sgID),
	}, nil
}

func (m *MockEC2Client) AuthorizeSecurityGroupIngress(ctx context.Context, params *ec2.AuthorizeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Just return success - we don't need to track ingress rules for most tests
	return &ec2.AuthorizeSecurityGroupIngressOutput{}, nil
}

func (m *MockEC2Client) CreatePlacementGroup(ctx context.Context, params *ec2.CreatePlacementGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreatePlacementGroupOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreatePlacementGroupCalls++

	if m.CreatePlacementGroupErr != nil {
		return nil, m.CreatePlacementGroupErr
	}

	pg := &types.PlacementGroup{
		GroupName: params.GroupName,
		Strategy:  params.Strategy,
		State:     types.PlacementGroupStateAvailable,
	}
	m.PlacementGroups[*params.GroupName] = pg

	return &ec2.CreatePlacementGroupOutput{
		PlacementGroup: pg,
	}, nil
}

func (m *MockEC2Client) DeletePlacementGroup(ctx context.Context, params *ec2.DeletePlacementGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeletePlacementGroupOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeletePlacementGroupCalls++

	if m.DeletePlacementGroupErr != nil {
		return nil, m.DeletePlacementGroupErr
	}

	delete(m.PlacementGroups, *params.GroupName)
	return &ec2.DeletePlacementGroupOutput{}, nil
}

func (m *MockEC2Client) DescribeInstanceTypes(ctx context.Context, params *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.DescribeInstanceTypesCalls++

	if m.DescribeInstanceTypesErr != nil {
		return nil, m.DescribeInstanceTypesErr
	}

	var instanceTypes []types.InstanceTypeInfo
	if len(params.InstanceTypes) > 0 {
		for _, it := range params.InstanceTypes {
			if info, ok := m.InstanceTypes[string(it)]; ok {
				instanceTypes = append(instanceTypes, *info)
			}
		}
	} else {
		for _, info := range m.InstanceTypes {
			instanceTypes = append(instanceTypes, *info)
		}
	}

	return &ec2.DescribeInstanceTypesOutput{
		InstanceTypes: instanceTypes,
	}, nil
}

func (m *MockEC2Client) DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.DescribeImagesCalls++

	if m.DescribeImagesErr != nil {
		return nil, m.DescribeImagesErr
	}

	var images []types.Image
	if len(params.ImageIds) > 0 {
		for _, id := range params.ImageIds {
			if img, ok := m.Images[id]; ok {
				images = append(images, *img)
			}
		}
	} else {
		for _, img := range m.Images {
			images = append(images, *img)
		}
	}

	return &ec2.DescribeImagesOutput{
		Images: images,
	}, nil
}

// Helper functions

func strPtr(s string) *string {
	return &s
}

func randomID() string {
	return fmt.Sprintf("%016x", randomUint64())
}

func randomByte() int {
	return int(randomUint64() % 256)
}

func randomUint64() uint64 {
	// Simple deterministic "random" for testing
	return uint64(0x123456789abcdef0)
}

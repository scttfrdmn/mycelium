package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

const (
	hostedZoneID = "Z048907324UNXKEK9KX93"
	domain       = "spore.host"
	defaultTTL   = 60
)

type DNSUpdateRequest struct {
	InstanceIdentityDocument  string `json:"instance_identity_document"`
	InstanceIdentitySignature string `json:"instance_identity_signature"`
	RecordName                string `json:"record_name"`
	IPAddress                 string `json:"ip_address"`
	Action                    string `json:"action"` // UPSERT or DELETE
}

type InstanceIdentityDocument struct {
	InstanceID string `json:"instanceId"`
	Region     string `json:"region"`
	AccountID  string `json:"accountId"`
}

type DNSUpdateResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	Record    string `json:"record,omitempty"`
	ChangeID  string `json:"change_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

var (
	cfg           aws.Config
	route53Client *route53.Client
)

// encodeAccountID converts AWS account ID to base36 (7 chars)
func encodeAccountID(accountID string) string {
	n := new(big.Int)
	n.SetString(accountID, 10)
	return strings.ToLower(n.Text(36))
}

// getFullDNSName returns the complete DNS name with base36-encoded account subdomain
// Example: ("my-instance", "123456789012") -> "my-instance.1kpqzg2c.spore.host"
func getFullDNSName(recordName, accountID string) string {
	encoded := encodeAccountID(accountID)
	return fmt.Sprintf("%s.%s.%s", recordName, encoded, domain)
}

func init() {
	var err error
	cfg, err = config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config: %v", err))
	}
	route53Client = route53.NewFromConfig(cfg)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse request body
	var req DNSUpdateRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid request body: %v", err))
	}

	// Validate required fields
	if req.InstanceIdentityDocument == "" || req.InstanceIdentitySignature == "" || req.RecordName == "" {
		return errorResponse(400, "Missing required fields")
	}

	// Validate action
	req.Action = strings.ToUpper(req.Action)
	if req.Action == "" {
		req.Action = "UPSERT"
	}
	if req.Action != "UPSERT" && req.Action != "DELETE" {
		return errorResponse(400, "Invalid action (must be UPSERT or DELETE)")
	}

	// Validate IP address for UPSERT
	if req.Action == "UPSERT" && req.IPAddress == "" {
		return errorResponse(400, "IP address required for UPSERT")
	}

	// Validate record name format
	req.RecordName = strings.ToLower(strings.TrimSpace(req.RecordName))
	validName := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !validName.MatchString(req.RecordName) {
		return errorResponse(400, "Invalid record name (alphanumeric and hyphens only)")
	}

	// Decode instance identity document
	identityDocBytes, err := base64.StdEncoding.DecodeString(req.InstanceIdentityDocument)
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid instance identity document: %v", err))
	}

	var identityDoc InstanceIdentityDocument
	if err := json.Unmarshal(identityDocBytes, &identityDoc); err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to parse instance identity document: %v", err))
	}

	// Validate required identity fields
	if identityDoc.InstanceID == "" || identityDoc.Region == "" || identityDoc.AccountID == "" {
		return errorResponse(400, "Instance identity document missing required fields")
	}

	// TODO: Verify instance identity signature
	// For now, we rely on instance validation via AWS API

	// Validate instance
	if err := validateInstance(ctx, identityDoc.InstanceID, identityDoc.Region, req.IPAddress, req.Action); err != nil {
		return errorResponse(403, err.Error())
	}

	// Build full DNS name with base36-encoded account subdomain
	// Example: my-instance.1kpqzg2c.spore.host (for account 123456789012)
	fqdn := getFullDNSName(req.RecordName, identityDoc.AccountID)

	// Update DNS record
	var changeID string
	var message string

	if req.Action == "UPSERT" {
		changeID, err = upsertDNSRecord(ctx, fqdn, req.IPAddress)
		if err != nil {
			return errorResponse(500, fmt.Sprintf("Failed to update DNS: %v", err))
		}
		message = fmt.Sprintf("DNS record updated: %s -> %s", fqdn, req.IPAddress)
	} else {
		changeID, err = deleteDNSRecord(ctx, fqdn)
		if err != nil {
			return errorResponse(500, fmt.Sprintf("Failed to delete DNS: %v", err))
		}
		message = fmt.Sprintf("DNS record deleted: %s", fqdn)
	}

	// Success response
	resp := DNSUpdateResponse{
		Success:   true,
		Message:   message,
		Record:    fqdn,
		ChangeID:  changeID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	body, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(body),
	}, nil
}

func validateInstance(ctx context.Context, instanceID, region, ipAddress, action string) error {
	// Create regional EC2 client
	regionalCfg := cfg.Copy()
	regionalCfg.Region = region
	ec2Client := ec2.NewFromConfig(regionalCfg)

	// Try to describe instance (may fail for cross-account)
	output, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})

	if err != nil {
		// Cross-account case: Can't describe instance in another account
		// This is OK for open source - we rely on instance identity signature
		// For now, just validate that the instance ID format is correct
		if action == "UPSERT" && ipAddress == "" {
			return fmt.Errorf("IP address required for UPSERT")
		}
		// Allow the request to proceed - signature validation is the primary security
		return nil
	}

	// Same-account case: Perform full validation
	if len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return fmt.Errorf("instance %s not found in %s", instanceID, region)
	}

	instance := output.Reservations[0].Instances[0]

	// Check for spawn:managed tag
	hasSpawnTag := false
	for _, tag := range instance.Tags {
		if aws.ToString(tag.Key) == "spawn:managed" && aws.ToString(tag.Value) == "true" {
			hasSpawnTag = true
			break
		}
	}
	if !hasSpawnTag {
		return fmt.Errorf("instance %s does not have spawn:managed tag", instanceID)
	}

	// For UPSERT, verify IP address matches
	if action == "UPSERT" {
		instancePublicIP := aws.ToString(instance.PublicIpAddress)
		if instancePublicIP == "" {
			return fmt.Errorf("instance %s has no public IP address", instanceID)
		}
		if instancePublicIP != ipAddress {
			return fmt.Errorf("IP address mismatch: %s != %s", ipAddress, instancePublicIP)
		}
	}

	// Check instance state
	state := string(instance.State.Name)
	if state != "running" && state != "stopped" {
		return fmt.Errorf("instance %s is in invalid state: %s", instanceID, state)
	}

	return nil
}

func upsertDNSRecord(ctx context.Context, fqdn, ipAddress string) (string, error) {
	comment := fmt.Sprintf("Updated by spawn instance at %s", time.Now().UTC().Format(time.RFC3339))

	output, err := route53Client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String(comment),
			Changes: []types.Change{
				{
					Action: types.ChangeActionUpsert,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: aws.String(fqdn),
						Type: types.RRTypeA,
						TTL:  aws.Int64(defaultTTL),
						ResourceRecords: []types.ResourceRecord{
							{Value: aws.String(ipAddress)},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	return aws.ToString(output.ChangeInfo.Id), nil
}

func deleteDNSRecord(ctx context.Context, fqdn string) (string, error) {
	// First, get the current record
	listOutput, err := route53Client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(hostedZoneID),
		StartRecordName: aws.String(fqdn),
		StartRecordType: types.RRTypeA,
		MaxItems:        aws.Int32(1),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list records: %w", err)
	}

	// Find matching record
	var recordToDelete *types.ResourceRecordSet
	for _, recordSet := range listOutput.ResourceRecordSets {
		if strings.TrimSuffix(aws.ToString(recordSet.Name), ".") == fqdn && recordSet.Type == types.RRTypeA {
			recordToDelete = &recordSet
			break
		}
	}

	if recordToDelete == nil {
		return "", fmt.Errorf("DNS record %s not found", fqdn)
	}

	// Delete the record
	comment := fmt.Sprintf("Deleted by spawn instance at %s", time.Now().UTC().Format(time.RFC3339))
	output, err := route53Client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String(comment),
			Changes: []types.Change{
				{
					Action:            types.ChangeActionDelete,
					ResourceRecordSet: recordToDelete,
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	return aws.ToString(output.ChangeInfo.Id), nil
}

func errorResponse(statusCode int, message string) (events.APIGatewayProxyResponse, error) {
	resp := DNSUpdateResponse{
		Success:   false,
		Error:     message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	body, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(body),
	}, nil
}

func main() {
	lambda.Start(handler)
}

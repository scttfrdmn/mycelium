package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// getUserFromRequest extracts user identity from API Gateway request
// Returns: userID, accountID, accountBase36, error
func getUserFromRequest(ctx context.Context, cfg aws.Config, request events.APIGatewayProxyRequest) (string, string, string, error) {
	// For IAM authentication, the user identity is in the request context
	// API Gateway IAM authorizer populates this

	// Extract user ARN from request context
	var userID string

	// Try to get user ARN from Identity (IAM auth)
	if request.RequestContext.Identity.UserArn != "" {
		userID = request.RequestContext.Identity.UserArn
	} else if request.RequestContext.Identity.Caller != "" {
		userID = request.RequestContext.Identity.Caller
	}

	// Fallback: Try to get from Authorizer (Cognito or custom authorizer)
	if userID == "" {
		if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
			if sub, ok := claims["sub"].(string); ok {
				userID = sub
			}
		}
	}

	// Fallback: Use principal ID
	if userID == "" {
		if principalID, ok := request.RequestContext.Authorizer["principalId"].(string); ok {
			userID = principalID
		}
	}

	// If still no user ID, return error
	if userID == "" {
		return "", "", "", fmt.Errorf("unable to determine user identity from request")
	}

	// Check if user account info is cached in DynamoDB
	cached, err := getUserAccount(ctx, cfg, userID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to query user account cache: %w", err)
	}

	// If cached, update last access and return
	if cached != nil {
		// Update last access asynchronously (don't block on this)
		go updateLastAccess(context.Background(), cfg, userID)

		return userID, cached.AWSAccountID, cached.AccountBase36, nil
	}

	// Not cached - detect account ID using STS
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get caller identity: %w", err)
	}

	if identity.Account == nil {
		return "", "", "", fmt.Errorf("account ID not returned by STS")
	}

	accountID := *identity.Account
	accountBase36 := intToBase36(accountID)

	// Create cache entry
	email := "" // TODO: Extract from Cognito claims if available
	err = createUserAccount(ctx, cfg, userID, accountID, accountBase36, email)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to cache user account: %v\n", err)
	}

	return userID, accountID, accountBase36, nil
}

// portal-phone-home — the spore.host portal's BYOA onboarding registrar.
//
// When a user onboards their AWS account to the portal (either via the
// `spawn onboard` CLI wizard or the web CloudFormation quick-create), the newly
// created cross-account role "phones home" to this Lambda to register itself:
// {roleArn, externalId, region}. The portal then knows which role+ExternalId to
// assume into that account.
//
// SECURITY — SigV4-verified-principal model (mirrors dns-updater, spawn#173):
// the Function URL runs under AuthType: AWS_IAM, so every request that reaches
// this handler has already passed SigV4 verification and carries the VERIFIED
// caller account in requestContext.authorizer.iam. We trust THAT account, never
// anything in the body. The body's roleArn MUST belong to the verified account,
// or we reject — so a caller can only ever register a role in its own account.
// No shared secret, no allow-list to maintain, no spoofable claims.
//
// The registry schema and the account lifecycle state machine live in the shared
// accountlifecycle module, because portal-account-prober writes lifecycle
// transitions to the same table and ApplyProbes must have exactly one copy.
package main

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/spore-host/spore-host/lambda/accountlifecycle"
)

var reg *accountlifecycle.Registry

// roleArnRE extracts the account id from an IAM role ARN and validates shape.
var roleArnRE = regexp.MustCompile(`^arn:aws:iam::(\d{12}):role/.+$`)

func init() {
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}
	reg = accountlifecycle.NewRegistry(cfg)
}

// phoneHomeRequest is the onboarding payload. accountId is NOT trusted from the
// body — it's derived from the SigV4-verified caller and cross-checked against
// roleArn. externalId is the confused-deputy guard the portal presents when
// assuming the role. region is where the user launches.
type phoneHomeRequest struct {
	RoleArn    string `json:"roleArn"`
	ExternalId string `json:"externalId"`
	Region     string `json:"region"`
}

func handler(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	if request.RequestContext.HTTP.Method != "POST" {
		return errorResponse(405, "method not allowed")
	}

	// ── Authorize via the SigV4-verified caller account ──────────────────────
	// Can't occur without an IAM authorizer under AuthType: AWS_IAM, but reject
	// defensively.
	authz := request.RequestContext.Authorizer
	if authz == nil || authz.IAM == nil || authz.IAM.AccountID == "" {
		return errorResponse(403, "missing IAM authorizer (Function URL must be AuthType: AWS_IAM)")
	}
	verifiedAccount := authz.IAM.AccountID

	var req phoneHomeRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return errorResponse(400, "invalid JSON body")
	}

	acct, verr := validate(verifiedAccount, authz.IAM.UserARN, req)
	if verr != nil {
		log.Printf("rejecting registration for verified account %s: %s", verifiedAccount, verr.msg)
		return errorResponse(verr.status, verr.msg)
	}

	if err := reg.PutAccount(ctx, acct); err != nil {
		log.Printf("put account %s: %v", verifiedAccount, err)
		return errorResponse(500, "failed to persist registration")
	}
	log.Printf("registered account %s (role %s) by %s", verifiedAccount, acct.RoleArn, authz.IAM.UserARN)

	return jsonResponse(200, map[string]string{
		"status":    "registered",
		"accountId": verifiedAccount,
		"region":    acct.Region,
	})
}

// validationError carries an HTTP status + message for a rejected request.
type validationError struct {
	status int
	msg    string
}

// validate is the pure, AWS-free core: given the SigV4-VERIFIED caller account
// (never the body's), enforce that the request is well-formed and that its
// roleArn belongs to the verified account, and normalize into an Account to
// persist. This is where the security invariant lives, so it's unit-tested with
// no AWS at all.
func validate(verifiedAccount, callerARN string, req phoneHomeRequest) (*accountlifecycle.Account, *validationError) {
	m := roleArnRE.FindStringSubmatch(strings.TrimSpace(req.RoleArn))
	if m == nil {
		return nil, &validationError{400, "roleArn must be a valid IAM role ARN"}
	}
	// The invariant: the role's account MUST equal the verified caller account.
	// You can only register a role that lives in your own account.
	if m[1] != verifiedAccount {
		return nil, &validationError{403, "roleArn account does not match the authenticated caller account"}
	}
	if strings.TrimSpace(req.ExternalId) == "" {
		return nil, &validationError{400, "externalId is required"}
	}
	region := strings.TrimSpace(req.Region)
	if region == "" {
		region = "us-east-1"
	}
	return &accountlifecycle.Account{
		AccountID:    verifiedAccount,
		RoleArn:      strings.TrimSpace(req.RoleArn),
		ExternalId:   strings.TrimSpace(req.ExternalId),
		Region:       region,
		RegisteredBy: callerARN,
	}, nil
}

func jsonResponse(status int, body any) (events.LambdaFunctionURLResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return errorResponse(500, "failed to encode response")
	}
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(b),
	}, nil
}

func errorResponse(status int, msg string) (events.LambdaFunctionURLResponse, error) {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(b),
	}, nil
}

func main() {
	lambda.Start(handler)
}

###############################################################################
# dns-updater — OpenTofu module (spawn#173).
#
# Reconciles the hand-deployed `spawn-dns-updater` Lambda (Route53 record updater
# for instance DNS registration) and its execution role under OpenTofu, mirroring
# the spore-bot module's import-onto-live pattern. The live resources were created
# imperatively (scripts/deploy-custom-dns.sh); this module is `tofu import`ed onto
# them and must `tofu plan` to a near-zero diff (only additive managedby tags)
# before it owns anything — see README.md.
#
# This is step 0 of the #173 cutover (move the DNS updater off the spoofable
# instance-identity-document auth onto the Function URL's AuthType: AWS_IAM):
# codify the resource first so the later cross-account InvokeFunctionUrl grant and
# the AuthType flip are reviewable IaC changes, not console edits.
#
# Deliberately NOT managed here (same discipline as spore-bot):
#   - Function CODE: deployed via scripts/deploy-custom-dns.sh / update-function-code.
#     Tofu ignores code attributes so it never reverts a deploy.
#   - Function ENV vars: DOMAIN_ZONES maps domains→hosted-zone IDs and is managed
#     out-of-band; Tofu ignores `environment` so it isn't clobbered.
###############################################################################

terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region  = var.region
  profile = var.aws_profile
}

variable "region" {
  type    = string
  default = "us-east-1"
}

variable "aws_profile" {
  type    = string
  default = "spore-host-infra"
}

# enable_iam_invoke: add the AWS_IAM invoke grant for the Function URL (#173
# step 1). Default false so step 0 imports with ZERO new resources. When the
# cutover is ready, set true to add the grant (additive + inert while AuthType is
# NONE), then flip authorization_type to AWS_IAM (step 3).
#
# The grant uses Principal: "*" deliberately — NOT an enumerated account/role
# list. spawn launches instances in arbitrarily many user accounts whose spored
# roles are dynamically named (spawn-instance-<hash>), so enumerating principals
# here would mean per-account infra enrollment and would hit the ~20KB resource
# policy limit. Instead:
#   - AuthType: AWS_IAM already requires every caller to present valid SigV4 —
#     so "*" means "any AWS principal that signs", not "anonymous".
#   - the spored instance role grants ITSELF lambda:InvokeFunctionUrl on this
#     function in its own identity policy (spawn/pkg/aws/iam.go), so access is
#     controlled per-account with zero infra-side enrollment.
#   - the Lambda handler enforces the real authz: it derives the SigV4-VERIFIED
#     caller account from requestContext and only lets a caller write records
#     under base36(verifiedAccountID).<domain> — closing the cross-account
#     spoofing this issue is about, cryptographically, with no allow-list to
#     maintain and no per-region certs (the reason IAM auth beat PKCS#7).
variable "enable_iam_invoke" {
  type    = bool
  default = false
}

locals {
  account_id = "966362334030"
  fn_name    = "spawn-dns-updater"
  role_name  = "SpawnDNSLambdaExecutionRole"
  # The Route53 hosted zone the updater writes (from the live DOMAIN_ZONES env).
  hosted_zone_id = "Z0341053304H0DQXF6U4X"
  common_tags = {
    project   = "spore-host"
    component = "dns-updater"
    managedby = "opentofu"
  }
}

# ── Execution role ───────────────────────────────────────────────────────────

resource "aws_iam_role" "dns_updater" {
  name        = local.role_name
  description = "Lambda execution role for spawn DNS updater"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "basic_execution" {
  role       = aws_iam_role.dns_updater.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# EC2DescribePolicy — used by the legacy validateInstance() same-account path.
# (Step 4 of #173 removes the identity-doc/validateInstance logic; this policy can
# be trimmed then. Mirrored as-is now for a zero-diff import.)
resource "aws_iam_role_policy" "ec2_describe" {
  name = "EC2DescribePolicy"
  role = aws_iam_role.dns_updater.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["ec2:DescribeInstances", "ec2:DescribeTags"]
      Resource = "*"
    }]
  })
}

resource "aws_iam_role_policy" "route53" {
  name = "Route53DNSUpdate"
  role = aws_iam_role.dns_updater.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["route53:GetHostedZone", "route53:ListResourceRecordSets", "route53:ChangeResourceRecordSets"]
        Resource = "arn:aws:route53:::hostedzone/${local.hosted_zone_id}"
      },
      {
        Effect   = "Allow"
        Action   = ["route53:ListHostedZones"]
        Resource = "*"
      }
    ]
  })
}

# X-Ray write permission, paired with the function's tracing_config (Active).
resource "aws_iam_role_policy" "xray" {
  name = "XRayWrite"
  role = aws_iam_role.dns_updater.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["xray:PutTraceSegments", "xray:PutTelemetryRecords"]
      Resource = "*"
    }]
  })
}

# ── Lambda function ──────────────────────────────────────────────────────────
# Tofu owns the function's SHAPE (role, runtime, arch, memory, timeout) but
# ignores code + environment (deployed/managed out-of-band).

resource "aws_lambda_function" "dns_updater" {
  function_name = local.fn_name
  role          = aws_iam_role.dns_updater.arn
  runtime       = "provided.al2023"
  handler       = "bootstrap"
  architectures = ["x86_64"]
  memory_size   = 256
  timeout       = 30

  # End-to-end request tracing (Semgrep best-practice; paired with xray:Put* in
  # the role). Applies on the next apply — a benign addition to the live function
  # (same treatment as the spore-bot module).
  tracing_config {
    mode = "Active"
  }

  # Placeholder code reference; real code comes from deploy-custom-dns.sh. Ignored
  # below so a deploy is never reverted.
  filename = "${path.module}/placeholder.zip"

  lifecycle {
    ignore_changes = [
      filename,
      source_code_hash,
      s3_bucket,
      s3_key,
      s3_object_version,
      environment, # DOMAIN_ZONES — managed out-of-band
      layers,
    ]
  }

  tags = local.common_tags
}

# ── Function URL ──────────────────────────────────────────────────────────────
# The endpoint instances POST to (zqonqra6…lambda-url.us-east-1.on.aws). Its value
# is deterministic from function name + account + region, so it is preserved
# across this import. AuthType stays NONE until the #173 cutover flips it to
# AWS_IAM (step 3) — see README; the flip is a one-line change here but MUST be
# lockstep with fielding SigV4-signing instances or all DNS registration breaks.
resource "aws_lambda_function_url" "dns_updater" {
  function_name      = aws_lambda_function.dns_updater.function_name
  authorization_type = "NONE"
  cors {
    allow_methods = ["POST"]
    allow_origins = ["*"]
    allow_headers = ["content-type"]
  }
}

# Public invoke permission for the Function URL (NONE auth) — the live grant.
resource "aws_lambda_permission" "url_public" {
  statement_id           = "FunctionURLAllowPublicAccess"
  action                 = "lambda:InvokeFunctionUrl"
  function_name          = aws_lambda_function.dns_updater.function_name
  principal              = "*"
  function_url_auth_type = "NONE"
}

# ── IAM invoke grant (#173 step 1, additive) ─────────────────────────────────
# Allows any SigV4-signed caller to invoke the Function URL under AWS_IAM auth.
# Principal "*" here is gated by AuthType: AWS_IAM (every caller must sign) plus
# the caller's own identity-policy grant and the Lambda's verified-account
# namespacing — see the enable_iam_invoke variable comment for the full rationale.
# Additive and inert while AuthType is NONE, so it is safe to apply before the
# flip. Created only when enable_iam_invoke = true.
resource "aws_lambda_permission" "url_iam_invoke" {
  count = var.enable_iam_invoke ? 1 : 0

  statement_id           = "FunctionURLAllowIAMAccess"
  action                 = "lambda:InvokeFunctionUrl"
  function_name          = aws_lambda_function.dns_updater.function_name
  principal              = "*"
  function_url_auth_type = "AWS_IAM"
}

output "function_url" {
  value       = aws_lambda_function_url.dns_updater.function_url
  description = "DNS updater Function URL (instances' spawn DNS endpoint)."
}

output "function_arn" {
  value = aws_lambda_function.dns_updater.arn
}

output "role_arn" {
  value = aws_iam_role.dns_updater.arn
}

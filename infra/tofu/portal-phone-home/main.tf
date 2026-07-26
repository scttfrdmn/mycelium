###############################################################################
# portal-phone-home — OpenTofu module (spore.host portal, Slice 3).
#
# The BYOA onboarding registrar for the portal. When a user onboards their AWS
# account (via `spawn onboard` or the web CloudFormation quick-create), the newly
# created cross-account role phones home to this Lambda's Function URL to register
# {roleArn, externalId, region}; the portal then knows how to assume into that
# account. Backend for Slice 3 (CLI wizard) + Slice 4 (web quick-create).
#
# Runs in the INFRA account (966362334030), the control plane — same account as
# dns-updater/spore-bot. This module mirrors dns-updater's shape, but unlike that
# import-onto-live module this is a FRESH apply (new function, URL, execution
# role, and DynamoDB table). Function CODE + env are managed out-of-band (Makefile
# upload); Tofu owns the shape and ignore_changes covers code.
#
# SECURITY: the Function URL is AuthType: AWS_IAM (like dns-updater post-#173).
# Principal "*" on the invoke grant means "any SigV4-signed caller", NOT anonymous
# — the onboarding role in the user's account grants ITSELF invoke on this URL in
# its own identity policy (created by the CFN template / CLI wizard), and the
# handler enforces the real authz: it derives the SigV4-verified caller account
# and rejects any body whose roleArn account differs. No allow-list to maintain.
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

locals {
  fn_name    = "portal-phone-home"
  role_name  = "PortalPhoneHomeLambdaRole"
  table_name = "spore-portal-accounts"
  common_tags = {
    project   = "spore-host"
    component = "portal-phone-home"
    managedby = "opentofu"
  }
}

# ── DynamoDB: onboarded-account registry ─────────────────────────────────────
# accountId (the SigV4-verified caller account) is the partition key; one row per
# onboarded account, upserted on re-onboard. On-demand billing (sporadic writes).
resource "aws_dynamodb_table" "accounts" {
  name         = local.table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "accountId"

  attribute {
    name = "accountId"
    type = "S"
  }

  point_in_time_recovery {
    enabled = true
  }

  tags = local.common_tags
}

# ── Execution role ───────────────────────────────────────────────────────────
resource "aws_iam_role" "phone_home" {
  name        = local.role_name
  description = "Lambda execution role for the portal BYOA phone-home registrar"
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
  role       = aws_iam_role.phone_home.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Write/read the accounts table only.
resource "aws_iam_role_policy" "dynamo" {
  name = "PortalAccountsTableAccess"
  role = aws_iam_role.phone_home.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["dynamodb:PutItem", "dynamodb:GetItem"]
      Resource = aws_dynamodb_table.accounts.arn
    }]
  })
}

# ── Lambda function ──────────────────────────────────────────────────────────
# Tofu owns the shape (role, runtime, arch, memory, timeout, env). Code is
# deployed out-of-band via the Makefile (S3 upload + update-function-code); the
# lifecycle block ignores code attributes so a deploy is never reverted.
resource "aws_lambda_function" "phone_home" {
  function_name = local.fn_name
  role          = aws_iam_role.phone_home.arn
  runtime       = "provided.al2023"
  handler       = "bootstrap"
  architectures = ["arm64"]
  memory_size   = 128
  timeout       = 10

  filename = "${path.module}/placeholder.zip"

  environment {
    variables = {
      ACCOUNTS_TABLE = aws_dynamodb_table.accounts.name
    }
  }

  lifecycle {
    ignore_changes = [
      filename,
      source_code_hash,
      s3_bucket,
      s3_key,
      s3_object_version,
    ]
  }

  tags = local.common_tags
}

# ── Function URL (AuthType: AWS_IAM) ─────────────────────────────────────────
resource "aws_lambda_function_url" "phone_home" {
  function_name      = aws_lambda_function.phone_home.function_name
  authorization_type = "AWS_IAM"
  cors {
    allow_methods = ["POST"]
    allow_origins = ["*"]
    allow_headers = ["content-type"]
  }
}

# Any SigV4-signed caller may invoke under AWS_IAM. Access is really controlled by
# the onboarding role's own identity-policy grant + the handler's verified-account
# check — see the header comment and dns-updater's identical rationale.
resource "aws_lambda_permission" "url_iam_invoke" {
  statement_id           = "FunctionURLAllowIAMAccess"
  action                 = "lambda:InvokeFunctionUrl"
  function_name          = aws_lambda_function.phone_home.function_name
  principal              = "*"
  function_url_auth_type = "AWS_IAM"
}

# ── Outputs ────────────────────────────────────────────────────────────────────
output "function_url" {
  value       = aws_lambda_function_url.phone_home.function_url
  description = "Phone-home Function URL — onboarding roles POST their registration here."
}

output "function_arn" {
  value = aws_lambda_function.phone_home.arn
}

output "role_arn" {
  value       = aws_iam_role.phone_home.arn
  description = "The phone-home Lambda execution role ARN."
}

output "accounts_table" {
  value = aws_dynamodb_table.accounts.name
}

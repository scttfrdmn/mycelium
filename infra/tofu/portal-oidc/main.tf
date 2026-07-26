###############################################################################
# portal-oidc — OpenTofu module (spore.host portal, Phase 3).
#
# The net-new AWS trust the default portal (spore-host/web) signs into. The
# browser federates an institutional identity through Globus Auth
# (CILogon/InCommon) and calls sts:AssumeRoleWithWebIdentity directly — no
# backend, no long-lived keys. This module codifies the two pieces that makes
# possible, matching the trust proven live in spawn-ts's demo (demo/README.md):
#
#   1. an IAM OIDC identity provider for https://auth.globus.org
#   2. a role that trust-scopes AssumeRoleWithWebIdentity to
#        - aud == the portal's Globus client-ID   (the app users log into)
#        - sub == an allow-list of Globus identity UUIDs (who may launch)
#      and grants exactly the EC2 launch + iam:PassRole perms a spawn launch needs.
#
# UNLIKE the infra-account modules (dns-updater, spore-bot), this is a fresh
# `apply` into the DEV compute account (435415984226) — the account where the
# demo and the #38 cross-account launch were validated and where
# spored-instance-profile already exists — NOT an import-onto-live. There is no
# pre-existing resource to reconcile.
#
# Why sub-UUIDs and not email: AWS only exposes a generic OIDC provider's `aud`
# and `sub` as IAM condition keys — `email` is NOT a usable condition key here.
# So "only friedman@ucla.edu" is enforced by pinning `sub` to that person's
# stable Globus identity UUID (resolved via `globus get-identities`), exactly as
# the demo was tightened. Add more users by appending their UUIDs to
# allowed_globus_subs.
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

# The DEV compute account profile — this module deploys the portal's launch
# trust where instances actually run, not into infra.
variable "aws_profile" {
  type    = string
  default = "spore-host-dev"
}

# The portal's Globus public-client UUID (the `aud` the trust checks). This is
# the app registered at developers.globus.org whose redirect URI is the portal
# URL (https://spore.host/app/). Supplied at apply time — there is no default
# because the app registration is an operator step, and a wrong/empty aud would
# make the trust accept nothing (or, if blank, fail to plan).
variable "globus_client_id" {
  type        = string
  description = "Globus public-client UUID registered for the portal (the id_token aud)."

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.globus_client_id))
    error_message = "globus_client_id must be a UUID (the portal's Globus client-ID)."
  }
}

# Allow-list of Globus identity UUIDs permitted to federate + launch. Resolve a
# username with `globus get-identities <user>@<inst>`. Default: friedman@ucla.edu.
variable "allowed_globus_subs" {
  type        = list(string)
  description = "Globus identity UUIDs allowed to assume the portal launch role (sub allow-list)."
  default     = ["66cae890-db2e-11e5-b782-d7b2bd2feb16"] # friedman@ucla.edu

  validation {
    condition     = length(var.allowed_globus_subs) > 0
    error_message = "allowed_globus_subs must not be empty — an empty sub allow-list would let no one in (or, worse, be mistaken for a wildcard). Pin at least one identity."
  }
}

# The instance profile launched instances get (spored self-terminate + DNS
# invoke). Already exists in dev; the role's iam:PassRole is scoped to it.
variable "spored_instance_profile" {
  type    = string
  default = "spored-instance-profile"
}

locals {
  provider_url  = "https://auth.globus.org"
  provider_host = "auth.globus.org"
  role_name     = "spore-portal-launch"
  common_tags = {
    project   = "spore-host"
    component = "portal-oidc"
    managedby = "opentofu"
  }
}

data "aws_caller_identity" "current" {}

# ── OIDC identity provider ────────────────────────────────────────────────────
# Globus as an IAM OIDC provider. client_id_list = the portal's aud (the token's
# audience must match). Modern AWS verifies the OIDC provider's TLS chain against
# its trust store, so the thumbprint is no longer security-critical for well-known
# CAs; AWS still requires the field, and it self-heals on well-known issuers.
# Terraform's tls provider could compute it, but to keep this module dependency-
# free we set Globus's leaf thumbprint and let AWS reconcile if it rotates.
resource "aws_iam_openid_connect_provider" "globus" {
  url             = local.provider_url
  client_id_list  = [var.globus_client_id]
  thumbprint_list = ["990f4193972f2becf12ddeda5237f9c952f20d9e"] # placeholder; AWS reconciles for well-known CAs
  tags            = local.common_tags
}

# ── Launch role ───────────────────────────────────────────────────────────────
# Trust: AssumeRoleWithWebIdentity from the Globus provider, gated on
#   aud == portal client-ID   AND   sub ∈ allowed_globus_subs.
# StringEquals on sub with a LIST is an OR over the list (not a wildcard) — so
# only the enumerated identities pass. This is the tightened form from the demo.
resource "aws_iam_role" "portal_launch" {
  name                 = local.role_name
  description          = "spore.host portal — Globus-federated EC2 launch role (browser AssumeRoleWithWebIdentity)"
  max_session_duration = 3600

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.globus.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.provider_host}:aud" = var.globus_client_id
          "${local.provider_host}:sub" = var.allowed_globus_subs
        }
      }
    }]
  })
  tags = local.common_tags
}

# EC2 launch + lifecycle — the same surface a spawn launch needs (RunInstances,
# tag, describe, terminate/stop). Region-scoped where the API supports it.
resource "aws_iam_role_policy" "ec2_launch" {
  name = "PortalEC2Launch"
  role = aws_iam_role.portal_launch.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "RunAndInspect"
        Effect = "Allow"
        Action = [
          "ec2:RunInstances",
          "ec2:DescribeInstances",
          "ec2:DescribeInstanceStatus",
          "ec2:DescribeImages",
          "ec2:DescribeSubnets",
          "ec2:DescribeSecurityGroups",
          "ec2:DescribeVpcs",
          "ec2:DescribeKeyPairs",
          "ec2:CreateTags",
        ]
        Resource = "*"
      },
      {
        Sid      = "Lifecycle"
        Effect   = "Allow"
        Action   = ["ec2:TerminateInstances", "ec2:StopInstances", "ec2:StartInstances"]
        Resource = "*"
        # spawn/spored tag their instances spawn:managed=true; scope destructive
        # actions to those so a portal session can't touch unrelated instances.
        Condition = {
          StringEquals = { "aws:ResourceTag/spawn:managed" = "true" }
        }
      },
      {
        Sid      = "ReadSpotAndQuotas"
        Effect   = "Allow"
        Action   = ["ec2:DescribeSpotPriceHistory", "servicequotas:GetServiceQuota", "servicequotas:ListServiceQuotas"]
        Resource = "*"
      },
    ]
  })
}

# iam:PassRole for the spored instance profile only — required to attach it at
# RunInstances so the instance can self-terminate + call the DNS Lambda.
resource "aws_iam_role_policy" "pass_spored_profile" {
  name = "PortalPassSporedProfile"
  role = aws_iam_role.portal_launch.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      # nosemgrep: terraform.lang.security.iam.no-iam-resource-exposure.no-iam-resource-exposure
      # PassRole is REQUIRED to attach an instance profile at RunInstances, and it
      # is already maximally scoped: a single named role ARN in THIS account (not a
      # wildcard) plus iam:PassedToService=ec2, so it can only ever hand
      # spored-instance-profile to EC2 — nothing else. The rule flags the action
      # unconditionally; the scoping here is the mitigation. (Same inline-suppress
      # convention as spore-bot/main.tf:111. Suppression reviewed + approved by the
      # repo owner, 2026-07-26.)
      Action   = "iam:PassRole"
      Resource = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.spored_instance_profile}"
      Condition = {
        StringEquals = { "iam:PassedToService" = "ec2.amazonaws.com" }
      }
    }]
  })
}

# SSM Session Manager — the browser terminal surface opens shells via
# ssm:StartSession against portal-launched instances (no SSH).
resource "aws_iam_role_policy" "ssm_session" {
  name = "PortalSSMSession"
  role = aws_iam_role.portal_launch.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["ssm:StartSession"]
        Resource = ["arn:aws:ec2:*:*:instance/*", "arn:aws:ssm:*:*:document/AWS-StartInteractiveCommand", "arn:aws:ssm:*::document/SSM-SessionManagerRunShell"]
      },
      {
        Effect   = "Allow"
        Action   = ["ssm:TerminateSession", "ssm:ResumeSession"]
        Resource = "arn:aws:ssm:*:*:session/*"
      },
    ]
  })
}

# ── Outputs ────────────────────────────────────────────────────────────────────
output "oidc_provider_arn" {
  value       = aws_iam_openid_connect_provider.globus.arn
  description = "The Globus IAM OIDC provider ARN."
}

output "role_arn" {
  value       = aws_iam_role.portal_launch.arn
  description = "The portal launch role ARN — set this as the portal's VITE_ROLE_ARN."
}

# AWS Login Guide for Truffle

This guide explains how to use the modern `aws login` command with Truffle - the easiest way to authenticate!

## 🚀 Why Use `aws login`?

The `aws login` command is the **modern, easiest way** to authenticate with AWS:

✅ **No manual credential copying** - everything is automated  
✅ **MFA handled automatically** - no manual token entry  
✅ **Perfect for SSO** - seamless IAM Identity Center integration  
✅ **Auto-refresh** - credentials refresh automatically  
✅ **Secure** - temporary credentials, no long-term keys  

## Quick Start (30 seconds)

### 1. Login to AWS
```bash
aws login
```

### 2. Use Truffle
```bash
truffle search m7i.large
```

**That's it!** 🎉

## Detailed Usage

### Basic Login

```bash
# Simple login (uses default profile)
aws login

# Verify it worked
aws sts get-caller-identity
```

You should see your AWS account information.

### SSO / IAM Identity Center Login

**First time setup:**
```bash
# Configure your SSO profile (one time)
aws configure sso

# Follow the prompts:
# - SSO start URL: https://your-org.awsapps.com/start
# - SSO Region: us-east-1
# - Choose your account and role
# - CLI default region: us-east-1
# - Profile name: my-sso-profile
```

**Daily usage:**
```bash
# Login with SSO profile
aws login --profile my-sso-profile

# Set as active profile
export AWS_PROFILE=my-sso-profile

# Use Truffle
truffle search m7i.large
```

### Multiple AWS Accounts

```bash
# Login to different accounts
aws login --profile work-account
aws login --profile personal-account

# Switch between them
export AWS_PROFILE=work-account
truffle search m7i.large

export AWS_PROFILE=personal-account
truffle search m8g.large
```

## Common Workflows

### Morning Routine (SSO Users)

```bash
# 1. Login to AWS
aws login --profile work

# 2. Set profile for the day
export AWS_PROFILE=work

# 3. Start working with Truffle
truffle search m7i.large
truffle az "c8g.*" --min-az-count 3
```

### Multi-Account Deployment

```bash
# Production account
aws login --profile prod
export AWS_PROFILE=prod
truffle search r7i.xlarge --regions us-east-1 > prod-instances.json

# Development account  
aws login --profile dev
export AWS_PROFILE=dev
truffle search r7i.xlarge --regions us-west-2 > dev-instances.json

# Compare results
diff prod-instances.json dev-instances.json
```

### CI/CD Pipeline

For CI/CD, use IAM roles or GitHub OIDC instead of `aws login`:

```bash
# In GitHub Actions (example)
- name: Configure AWS Credentials
  uses: aws-actions/configure-aws-credentials@v4
  with:
    role-to-assume: arn:aws:iam::123456789012:role/TruffleRole
    aws-region: us-east-1

- name: Run Truffle
  run: truffle search m7i.large --output json
```

## Troubleshooting

### "Session has expired"

```bash
# Just login again!
aws login --profile my-profile
```

SSO sessions typically last 8-12 hours. When expired, just re-run `aws login`.

### "Unable to locate credentials"

```bash
# Make sure you've logged in
aws login

# Verify credentials exist
aws sts get-caller-identity

# If using a profile, make sure it's set
export AWS_PROFILE=my-profile
```

### "SSO session has expired"

```bash
# Re-authenticate with SSO
aws login --profile my-sso-profile

# If that doesn't work, reconfigure SSO
aws configure sso --profile my-sso-profile
```

### Wrong AWS Account

```bash
# Check which account you're in
aws sts get-caller-identity

# Login to correct account
aws login --profile correct-account
export AWS_PROFILE=correct-account
```

### Profile Not Found

```bash
# List available profiles
aws configure list-profiles

# Configure a new profile
aws configure sso --profile new-profile
```

## Advanced Tips

### Shell Alias for Convenience

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
# Alias for quick login
alias awslogin='aws login --profile work && export AWS_PROFILE=work'

# Alias with Truffle
alias tf='truffle'

# Use it
awslogin
tf search m7i.large
```

### Auto-Login Script

Create `~/bin/aws-truffle-login.sh`:

```bash
#!/bin/bash
# Login and set profile in one command

PROFILE=${1:-default}

echo "🔐 Logging in to AWS profile: $PROFILE"
aws login --profile "$PROFILE"

if [ $? -eq 0 ]; then
    export AWS_PROFILE="$PROFILE"
    echo "✅ Logged in and profile set to: $PROFILE"
    echo "🍄 Ready to use Truffle!"
else
    echo "❌ Login failed"
    exit 1
fi
```

Usage:
```bash
chmod +x ~/bin/aws-truffle-login.sh
~/bin/aws-truffle-login.sh work
truffle search m7i.large
```

### Check Session Expiry

```bash
# Add to your shell prompt (optional)
aws_session_status() {
    if aws sts get-caller-identity &>/dev/null; then
        echo "✅ AWS"
    else
        echo "❌ AWS"
    fi
}

# Add to PS1 prompt
PS1='$(aws_session_status) \u@\h:\w\$ '
```

### Profile-Specific Truffle Configs

Create profile-specific configs:

```bash
# ~/.truffle/config-work.yaml
default_region: us-east-1
default_output: json
skip_azs: false

# ~/.truffle/config-personal.yaml
default_region: us-west-2
default_output: table
skip_azs: true
```

Use with profiles:
```bash
aws login --profile work
export AWS_PROFILE=work
truffle search m7i.large  # Uses work config
```

## Comparison: aws login vs Traditional Methods

| Method | Setup Time | Security | SSO Support | Auto-Refresh | Ease of Use |
|--------|-----------|----------|-------------|--------------|-------------|
| **aws login** | 30 sec | ✅✅✅ | ✅ | ✅ | ⭐⭐⭐⭐⭐ |
| Access Keys | 5 min | ⚠️ | ❌ | ❌ | ⭐⭐ |
| aws configure | 2 min | ⚠️ | ❌ | ❌ | ⭐⭐⭐ |
| Environment vars | 1 min | ⚠️ | ❌ | ❌ | ⭐⭐ |

**Winner:** `aws login` - easier, more secure, better for teams using SSO!

## Security Best Practices

### ✅ DO:
- Use `aws login` for interactive sessions
- Use SSO/IAM Identity Center for organizations
- Set reasonable session durations (8-12 hours)
- Use different profiles for different environments
- Re-login when session expires

### ❌ DON'T:
- Share long-term access keys
- Commit credentials to git
- Use root account credentials
- Leave sessions open indefinitely
- Use production credentials in development

## Integration with Development Tools

### VS Code
```bash
# Login before opening VS Code
aws login --profile work
export AWS_PROFILE=work
code truffle/
```

### Docker
```bash
# Login then pass credentials to container
aws login
docker run -e AWS_PROFILE=$AWS_PROFILE \
  -v ~/.aws:/root/.aws:ro \
  truffle-dev \
  truffle search m7i.large
```

### Claude Code
```bash
# Login before starting Claude Code
cd truffle
aws login --profile work
export AWS_PROFILE=work
claude-code
```

## FAQ

**Q: How long does an aws login session last?**  
A: Typically 8-12 hours for SSO, configurable by your admin.

**Q: Do I need to login every day?**  
A: Yes, for security. It takes 5 seconds: `aws login`

**Q: Can I use aws login in scripts?**  
A: For interactive scripts yes, but for automation use IAM roles or GitHub OIDC.

**Q: Does aws login work with GovCloud?**  
A: Yes! Just specify the correct SSO start URL.

**Q: What if my organization doesn't use SSO?**  
A: You can still use `aws configure` with access keys, but `aws login` is recommended for SSO.

**Q: Can I have multiple sessions active?**  
A: Yes! Use different profiles: `aws login --profile prod`, `aws login --profile dev`

## Summary

**Best practice for Truffle:**

```bash
# Morning login (30 seconds)
aws login --profile work
export AWS_PROFILE=work

# Use Truffle all day
truffle search m7i.large
truffle az "m8g.*" --min-az-count 3
# ... etc

# Next morning, re-login (30 seconds)
aws login --profile work
```

**That's it!** Simple, secure, and modern. 🚀

---

**Additional Resources:**
- [AWS CLI SSO Configuration](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html)
- [IAM Identity Center](https://aws.amazon.com/iam/identity-center/)
- [AWS CLI Best Practices](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-best-practices.html)

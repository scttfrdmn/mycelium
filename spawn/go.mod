module github.com/scttfrdmn/mycelium/spawn

go 1.24.0

require (
	github.com/aws/aws-sdk-go-v2 v1.41.1
	github.com/aws/aws-sdk-go-v2/config v1.26.6
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.11
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.141.0
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.1
	github.com/aws/aws-sdk-go-v2/service/ssm v1.44.0
	github.com/spf13/cobra v1.8.0
	golang.org/x/crypto v0.46.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.20.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.53.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.32.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/efs v1.41.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/fsx v1.65.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/lambda v1.87.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.95.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.7 // indirect
	github.com/aws/smithy-go v1.24.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/nicksnyder/go-i18n/v2 v2.4.1 // indirect
	github.com/scttfrdmn/mycelium/pkg/i18n v0.0.0-00010101000000-000000000000 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)

replace github.com/scttfrdmn/mycelium/pkg/i18n => ../pkg/i18n

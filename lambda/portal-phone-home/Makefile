.PHONY: build clean test upload

# Configurable — override with environment variables
S3_BUCKET     ?= spawn-binaries-us-east-1
FUNCTION_NAME ?= portal-phone-home
REGION        ?= us-east-1

## build: Compile Lambda binary and zip for deployment
build:
	@echo "Building portal-phone-home Lambda (linux/arm64)..."
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap .
	zip function.zip bootstrap
	@echo "Build complete: function.zip"

## clean: Remove build artifacts
clean:
	rm -f bootstrap function.zip

## test: Run unit tests
test:
	go test ./... -v

## upload: Build and upload zip to S3
upload: build
	@echo "Uploading to s3://$(S3_BUCKET)/portal-phone-home/function.zip..."
	AWS_PROFILE=spore-host-infra aws s3 cp function.zip \
		s3://$(S3_BUCKET)/portal-phone-home/function.zip \
		--region $(REGION)
	@echo "Upload complete"

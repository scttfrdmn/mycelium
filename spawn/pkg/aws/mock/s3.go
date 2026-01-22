package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3API defines the subset of S3 API operations we need to mock
type S3API interface {
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	PutBucketTagging(ctx context.Context, params *s3.PutBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.PutBucketTaggingOutput, error)
	PutBucketLifecycleConfiguration(ctx context.Context, params *s3.PutBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.PutBucketLifecycleConfigurationOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
}

// MockS3Client provides a mock implementation of S3 operations for testing
type MockS3Client struct {
	mu sync.RWMutex

	// Mock data storage
	Buckets map[string]*Bucket
	Objects map[string]map[string]*Object // bucket -> key -> object

	// Errors to return for specific operations
	CreateBucketErr                     error
	HeadBucketErr                       error
	PutBucketTaggingErr                 error
	PutBucketLifecycleConfigurationErr  error
	PutObjectErr                        error
	GetObjectErr                        error
	HeadObjectErr                       error
	ListObjectsV2Err                    error
	DeleteObjectErr                     error
	CopyObjectErr                       error

	// Call tracking
	CreateBucketCalls                     int
	HeadBucketCalls                       int
	PutBucketTaggingCalls                 int
	PutBucketLifecycleConfigurationCalls  int
	PutObjectCalls                        int
	GetObjectCalls                        int
	HeadObjectCalls                       int
	ListObjectsV2Calls                    int
	DeleteObjectCalls                     int
	CopyObjectCalls                       int
}

// Bucket represents a mock S3 bucket
type Bucket struct {
	Name                string
	Region              string
	Tags                []types.Tag
	LifecycleRules      []types.LifecycleRule
}

// Object represents a mock S3 object
type Object struct {
	Key          string
	Data         []byte
	Metadata     map[string]string
	ContentType  *string
	Size         int64
	ETag         *string
}

// NewMockS3Client creates a new mock S3 client
func NewMockS3Client() *MockS3Client {
	return &MockS3Client{
		Buckets: make(map[string]*Bucket),
		Objects: make(map[string]map[string]*Object),
	}
}

func (m *MockS3Client) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateBucketCalls++

	if m.CreateBucketErr != nil {
		return nil, m.CreateBucketErr
	}

	bucket := &Bucket{
		Name: *params.Bucket,
	}

	if params.CreateBucketConfiguration != nil && params.CreateBucketConfiguration.LocationConstraint != "" {
		bucket.Region = string(params.CreateBucketConfiguration.LocationConstraint)
	}

	m.Buckets[*params.Bucket] = bucket
	m.Objects[*params.Bucket] = make(map[string]*Object)

	return &s3.CreateBucketOutput{}, nil
}

func (m *MockS3Client) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.HeadBucketCalls++

	if m.HeadBucketErr != nil {
		return nil, m.HeadBucketErr
	}

	if _, ok := m.Buckets[*params.Bucket]; !ok {
		return nil, fmt.Errorf("bucket not found: %s", *params.Bucket)
	}

	return &s3.HeadBucketOutput{}, nil
}

func (m *MockS3Client) PutBucketTagging(ctx context.Context, params *s3.PutBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.PutBucketTaggingOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PutBucketTaggingCalls++

	if m.PutBucketTaggingErr != nil {
		return nil, m.PutBucketTaggingErr
	}

	bucket, ok := m.Buckets[*params.Bucket]
	if !ok {
		return nil, fmt.Errorf("bucket not found: %s", *params.Bucket)
	}

	bucket.Tags = params.Tagging.TagSet

	return &s3.PutBucketTaggingOutput{}, nil
}

func (m *MockS3Client) PutBucketLifecycleConfiguration(ctx context.Context, params *s3.PutBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.PutBucketLifecycleConfigurationOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PutBucketLifecycleConfigurationCalls++

	if m.PutBucketLifecycleConfigurationErr != nil {
		return nil, m.PutBucketLifecycleConfigurationErr
	}

	bucket, ok := m.Buckets[*params.Bucket]
	if !ok {
		return nil, fmt.Errorf("bucket not found: %s", *params.Bucket)
	}

	if params.LifecycleConfiguration != nil {
		bucket.LifecycleRules = params.LifecycleConfiguration.Rules
	}

	return &s3.PutBucketLifecycleConfigurationOutput{}, nil
}

func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PutObjectCalls++

	if m.PutObjectErr != nil {
		return nil, m.PutObjectErr
	}

	bucketObjects, ok := m.Objects[*params.Bucket]
	if !ok {
		return nil, fmt.Errorf("bucket not found: %s", *params.Bucket)
	}

	// Read data from Body (simplified - in reality would need proper io.Reader handling)
	data := []byte{}
	if params.Body != nil {
		// For testing, we'll just store an empty byte slice
		// In real tests, you'd read from the Body
	}

	etag := fmt.Sprintf("\"%s\"", randomID())
	obj := &Object{
		Key:         *params.Key,
		Data:        data,
		Metadata:    params.Metadata,
		ContentType: params.ContentType,
		Size:        int64(len(data)),
		ETag:        &etag,
	}

	bucketObjects[*params.Key] = obj

	return &s3.PutObjectOutput{
		ETag: &etag,
	}, nil
}

func (m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetObjectCalls++

	if m.GetObjectErr != nil {
		return nil, m.GetObjectErr
	}

	bucketObjects, ok := m.Objects[*params.Bucket]
	if !ok {
		return nil, fmt.Errorf("bucket not found: %s", *params.Bucket)
	}

	obj, ok := bucketObjects[*params.Key]
	if !ok {
		return nil, fmt.Errorf("object not found: %s/%s", *params.Bucket, *params.Key)
	}

	// In real tests, you'd return an io.ReadCloser with the object data
	return &s3.GetObjectOutput{
		ContentLength: &obj.Size,
		ContentType:   obj.ContentType,
		ETag:          obj.ETag,
		Metadata:      obj.Metadata,
	}, nil
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.HeadObjectCalls++

	if m.HeadObjectErr != nil {
		return nil, m.HeadObjectErr
	}

	bucketObjects, ok := m.Objects[*params.Bucket]
	if !ok {
		return nil, fmt.Errorf("bucket not found: %s", *params.Bucket)
	}

	obj, ok := bucketObjects[*params.Key]
	if !ok {
		return nil, fmt.Errorf("object not found: %s/%s", *params.Bucket, *params.Key)
	}

	return &s3.HeadObjectOutput{
		ContentLength: &obj.Size,
		ContentType:   obj.ContentType,
		ETag:          obj.ETag,
		Metadata:      obj.Metadata,
	}, nil
}

func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.ListObjectsV2Calls++

	if m.ListObjectsV2Err != nil {
		return nil, m.ListObjectsV2Err
	}

	bucketObjects, ok := m.Objects[*params.Bucket]
	if !ok {
		return nil, fmt.Errorf("bucket not found: %s", *params.Bucket)
	}

	var contents []types.Object
	for key, obj := range bucketObjects {
		// Apply prefix filter if provided
		if params.Prefix != nil && !matchPrefix(key, *params.Prefix) {
			continue
		}

		contents = append(contents, types.Object{
			Key:  &key,
			Size: &obj.Size,
			ETag: obj.ETag,
		})
	}

	keyCount := int32(len(contents))
	return &s3.ListObjectsV2Output{
		Contents: contents,
		KeyCount: &keyCount,
	}, nil
}

func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteObjectCalls++

	if m.DeleteObjectErr != nil {
		return nil, m.DeleteObjectErr
	}

	bucketObjects, ok := m.Objects[*params.Bucket]
	if !ok {
		return nil, fmt.Errorf("bucket not found: %s", *params.Bucket)
	}

	delete(bucketObjects, *params.Key)

	return &s3.DeleteObjectOutput{}, nil
}

func (m *MockS3Client) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CopyObjectCalls++

	if m.CopyObjectErr != nil {
		return nil, m.CopyObjectErr
	}

	// Parse source bucket and key from CopySource
	// Format: /bucket/key
	// Simplified parsing for mock
	destBucket := *params.Bucket
	destKey := *params.Key

	destBucketObjects, ok := m.Objects[destBucket]
	if !ok {
		return nil, fmt.Errorf("destination bucket not found: %s", destBucket)
	}

	// For simplicity, create a new object
	etag := fmt.Sprintf("\"%s\"", randomID())
	obj := &Object{
		Key:  destKey,
		Data: []byte{},
		Size: 0,
		ETag: &etag,
	}

	destBucketObjects[destKey] = obj

	return &s3.CopyObjectOutput{
		CopyObjectResult: &types.CopyObjectResult{
			ETag: &etag,
		},
	}, nil
}

// Helper functions

func matchPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

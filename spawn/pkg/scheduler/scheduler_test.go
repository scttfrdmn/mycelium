package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
)

// Mock DynamoDB client
type mockDynamoDBClient struct {
	getItemFunc    func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	putItemFunc    func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	updateItemFunc func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	deleteItemFunc func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	queryFunc      func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

func (m *mockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if m.getItemFunc != nil {
		return m.getItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.GetItemOutput{}, nil
}

func (m *mockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if m.putItemFunc != nil {
		return m.putItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.PutItemOutput{}, nil
}

func (m *mockDynamoDBClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	if m.updateItemFunc != nil {
		return m.updateItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.UpdateItemOutput{}, nil
}

func (m *mockDynamoDBClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	if m.deleteItemFunc != nil {
		return m.deleteItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.DeleteItemOutput{}, nil
}

func (m *mockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, params, optFns...)
	}
	return &dynamodb.QueryOutput{}, nil
}

// Mock Scheduler client
type mockSchedulerClient struct {
	createScheduleFunc func(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error)
	deleteScheduleFunc func(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error)
	updateScheduleFunc func(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error)
}

func (m *mockSchedulerClient) CreateSchedule(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error) {
	if m.createScheduleFunc != nil {
		return m.createScheduleFunc(ctx, params, optFns...)
	}
	return &scheduler.CreateScheduleOutput{
		ScheduleArn: aws.String("arn:aws:scheduler:us-east-1:123456789012:schedule/test-schedule"),
	}, nil
}

func (m *mockSchedulerClient) DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error) {
	if m.deleteScheduleFunc != nil {
		return m.deleteScheduleFunc(ctx, params, optFns...)
	}
	return &scheduler.DeleteScheduleOutput{}, nil
}

func (m *mockSchedulerClient) UpdateSchedule(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error) {
	if m.updateScheduleFunc != nil {
		return m.updateScheduleFunc(ctx, params, optFns...)
	}
	return &scheduler.UpdateScheduleOutput{}, nil
}

func TestCreateSchedule(t *testing.T) {
	tests := []struct {
		name       string
		record     *ScheduleRecord
		wantErr    bool
		setupMocks func(*mockDynamoDBClient, *mockSchedulerClient)
	}{
		{
			name: "one-time schedule",
			record: &ScheduleRecord{
				ScheduleID:         "sched-test-001",
				UserID:             "user-123",
				ScheduleName:       "test-schedule",
				ScheduleExpression: "at(2026-01-22T15:00:00)",
				ScheduleType:       "one-time",
				Timezone:           "America/New_York",
				S3ParamsKey:        "schedules/sched-test-001/params.yaml",
				SweepName:          "test-sweep",
				Status:             "active",
				MaxConcurrent:      10,
				Region:             "us-east-1",
			},
			wantErr: false,
			setupMocks: func(ddb *mockDynamoDBClient, sched *mockSchedulerClient) {
				ddb.putItemFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					if params.TableName == nil || *params.TableName != "spawn-schedules" {
						t.Errorf("Expected table name spawn-schedules, got %v", params.TableName)
					}
					return &dynamodb.PutItemOutput{}, nil
				}
				sched.createScheduleFunc = func(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error) {
					if params.ScheduleExpression == nil || *params.ScheduleExpression != "at(2026-01-22T15:00:00)" {
						t.Errorf("Expected schedule expression at(2026-01-22T15:00:00), got %v", params.ScheduleExpression)
					}
					return &scheduler.CreateScheduleOutput{
						ScheduleArn: aws.String("arn:aws:scheduler:us-east-1:123456789012:schedule/sched-test-001"),
					}, nil
				}
			},
		},
		{
			name: "recurring schedule with cron",
			record: &ScheduleRecord{
				ScheduleID:         "sched-test-002",
				UserID:             "user-123",
				ScheduleName:       "nightly-training",
				ScheduleExpression: "cron(0 2 * * ? *)",
				ScheduleType:       "recurring",
				Timezone:           "America/New_York",
				S3ParamsKey:        "schedules/sched-test-002/params.yaml",
				SweepName:          "nightly-sweep",
				Status:             "active",
				MaxExecutions:      30,
				Region:             "us-east-1",
			},
			wantErr: false,
			setupMocks: func(ddb *mockDynamoDBClient, sched *mockSchedulerClient) {
				ddb.putItemFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					return &dynamodb.PutItemOutput{}, nil
				}
				sched.createScheduleFunc = func(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error) {
					if params.ScheduleExpression == nil || *params.ScheduleExpression != "cron(0 2 * * ? *)" {
						t.Errorf("Expected cron expression, got %v", params.ScheduleExpression)
					}
					return &scheduler.CreateScheduleOutput{
						ScheduleArn: aws.String("arn:aws:scheduler:us-east-1:123456789012:schedule/sched-test-002"),
					}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDDB := &mockDynamoDBClient{}
			mockSched := &mockSchedulerClient{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockDDB, mockSched)
			}

			client := &Client{
				dynamoClient:    mockDDB,
				schedulerClient: mockSched,
				lambdaARN:       "arn:aws:lambda:us-east-1:123456789012:function:scheduler-handler",
				roleARN:         "arn:aws:iam::123456789012:role/EventBridgeSchedulerRole",
				tableName:       "spawn-schedules",
			}

			_, err := client.CreateSchedule(context.Background(), tt.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetSchedule(t *testing.T) {
	tests := []struct {
		name       string
		scheduleID string
		setupMock  func(*mockDynamoDBClient)
		wantErr    bool
	}{
		{
			name:       "existing schedule",
			scheduleID: "sched-test-001",
			setupMock: func(ddb *mockDynamoDBClient) {
				ddb.getItemFunc = func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: map[string]dynamodbtypes.AttributeValue{
							"schedule_id":         &dynamodbtypes.AttributeValueMemberS{Value: "sched-test-001"},
							"user_id":             &dynamodbtypes.AttributeValueMemberS{Value: "user-123"},
							"schedule_name":       &dynamodbtypes.AttributeValueMemberS{Value: "test-schedule"},
							"schedule_expression": &dynamodbtypes.AttributeValueMemberS{Value: "at(2026-01-22T15:00:00)"},
							"schedule_type":       &dynamodbtypes.AttributeValueMemberS{Value: "one-time"},
							"timezone":            &dynamodbtypes.AttributeValueMemberS{Value: "America/New_York"},
							"status":              &dynamodbtypes.AttributeValueMemberS{Value: "active"},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:       "non-existent schedule",
			scheduleID: "sched-missing",
			setupMock: func(ddb *mockDynamoDBClient) {
				ddb.getItemFunc = func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					return &dynamodb.GetItemOutput{
						Item: nil,
					}, nil
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDDB := &mockDynamoDBClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockDDB)
			}

			client := &Client{
				dynamoClient: mockDDB,
				tableName:    "spawn-schedules",
			}

			record, err := client.GetSchedule(context.Background(), tt.scheduleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && record == nil {
				t.Error("GetSchedule() returned nil record when error not expected")
			}
		})
	}
}

func TestDeleteSchedule(t *testing.T) {
	tests := []struct {
		name       string
		scheduleID string
		setupMocks func(*mockDynamoDBClient, *mockSchedulerClient)
		wantErr    bool
	}{
		{
			name:       "successful deletion",
			scheduleID: "sched-test-001",
			setupMocks: func(ddb *mockDynamoDBClient, sched *mockSchedulerClient) {
				ddb.deleteItemFunc = func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
					return &dynamodb.DeleteItemOutput{}, nil
				}
				sched.deleteScheduleFunc = func(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error) {
					if params.Name == nil || *params.Name != "sched-test-001" {
						t.Errorf("Expected schedule name sched-test-001, got %v", params.Name)
					}
					return &scheduler.DeleteScheduleOutput{}, nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDDB := &mockDynamoDBClient{}
			mockSched := &mockSchedulerClient{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockDDB, mockSched)
			}

			client := &Client{
				dynamoClient:    mockDDB,
				schedulerClient: mockSched,
				tableName:       "spawn-schedules",
			}

			err := client.DeleteSchedule(context.Background(), tt.scheduleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateScheduleStatus(t *testing.T) {
	tests := []struct {
		name       string
		scheduleID string
		status     ScheduleStatus
		setupMocks func(*mockDynamoDBClient, *mockSchedulerClient)
		wantErr    bool
	}{
		{
			name:       "pause schedule",
			scheduleID: "sched-test-001",
			status:     ScheduleStatusPaused,
			setupMocks: func(ddb *mockDynamoDBClient, sched *mockSchedulerClient) {
				ddb.updateItemFunc = func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					return &dynamodb.UpdateItemOutput{}, nil
				}
				sched.updateScheduleFunc = func(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error) {
					if params.State != schedulertypes.ScheduleStateDisabled {
						t.Error("Expected schedule state to be DISABLED")
					}
					return &scheduler.UpdateScheduleOutput{}, nil
				}
			},
			wantErr: false,
		},
		{
			name:       "resume schedule",
			scheduleID: "sched-test-001",
			status:     ScheduleStatusActive,
			setupMocks: func(ddb *mockDynamoDBClient, sched *mockSchedulerClient) {
				ddb.updateItemFunc = func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					return &dynamodb.UpdateItemOutput{}, nil
				}
				sched.updateScheduleFunc = func(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error) {
					if params.State != schedulertypes.ScheduleStateEnabled {
						t.Error("Expected schedule state to be ENABLED")
					}
					return &scheduler.UpdateScheduleOutput{}, nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDDB := &mockDynamoDBClient{}
			mockSched := &mockSchedulerClient{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockDDB, mockSched)
			}

			client := &Client{
				dynamoClient:    mockDDB,
				schedulerClient: mockSched,
				tableName:       "spawn-schedules",
			}

			err := client.UpdateScheduleStatus(context.Background(), tt.scheduleID, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateScheduleStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecordExecution(t *testing.T) {
	tests := []struct {
		name       string
		scheduleID string
		sweepID    string
		status     string
		setupMock  func(*mockDynamoDBClient)
		wantErr    bool
	}{
		{
			name:       "successful execution record",
			scheduleID: "sched-test-001",
			sweepID:    "sweep-20260122-140530",
			status:     "success",
			setupMock: func(ddb *mockDynamoDBClient) {
				ddb.putItemFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					if params.TableName == nil || *params.TableName != "spawn-schedule-history" {
						t.Errorf("Expected table name spawn-schedule-history, got %v", params.TableName)
					}
					return &dynamodb.PutItemOutput{}, nil
				}
			},
			wantErr: false,
		},
		{
			name:       "failed execution record",
			scheduleID: "sched-test-002",
			sweepID:    "sweep-20260122-150530",
			status:     "failed",
			setupMock: func(ddb *mockDynamoDBClient) {
				ddb.putItemFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					return &dynamodb.PutItemOutput{}, nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDDB := &mockDynamoDBClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockDDB)
			}

			client := &Client{
				dynamoClient: mockDDB,
				tableName:    "spawn-schedules",
			}

			history := &ExecutionHistory{
				ScheduleID:    tt.scheduleID,
				ExecutionTime: time.Now(),
				SweepID:       tt.sweepID,
				Status:        tt.status,
			}

			err := client.RecordExecution(context.Background(), history)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordExecution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduleRecordValidation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		record *ScheduleRecord
		valid  bool
	}{
		{
			name: "valid one-time schedule",
			record: &ScheduleRecord{
				ScheduleID:         "sched-test-001",
				ScheduleType:       "one-time",
				ScheduleExpression: "at(2026-01-22T15:00:00)",
				Status:             "active",
			},
			valid: true,
		},
		{
			name: "valid recurring schedule",
			record: &ScheduleRecord{
				ScheduleID:         "sched-test-002",
				ScheduleType:       "recurring",
				ScheduleExpression: "cron(0 2 * * ? *)",
				Status:             "active",
			},
			valid: true,
		},
		{
			name: "recurring with max executions",
			record: &ScheduleRecord{
				ScheduleID:         "sched-test-003",
				ScheduleType:       "recurring",
				ScheduleExpression: "cron(0 2 * * ? *)",
				Status:             "active",
				MaxExecutions:      30,
			},
			valid: true,
		},
		{
			name: "recurring with end date",
			record: &ScheduleRecord{
				ScheduleID:         "sched-test-004",
				ScheduleType:       "recurring",
				ScheduleExpression: "cron(0 2 * * ? *)",
				Status:             "active",
				EndAfter:           now.Add(30 * 24 * time.Hour),
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			if tt.record.ScheduleID == "" {
				t.Error("ScheduleID should not be empty")
			}
			if tt.record.ScheduleType != "one-time" && tt.record.ScheduleType != "recurring" {
				t.Error("ScheduleType must be one-time or recurring")
			}
			if tt.record.Status != "active" && tt.record.Status != "paused" && tt.record.Status != "cancelled" {
				t.Error("Invalid status")
			}
		})
	}
}

func TestListSchedulesByUser(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(*mockDynamoDBClient)
		wantCount int
		wantErr   bool
	}{
		{
			name:   "user with schedules",
			userID: "user-123",
			setupMock: func(ddb *mockDynamoDBClient) {
				ddb.queryFunc = func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{
						Items: []map[string]dynamodbtypes.AttributeValue{
							{
								"schedule_id":         &dynamodbtypes.AttributeValueMemberS{Value: "sched-test-001"},
								"user_id":             &dynamodbtypes.AttributeValueMemberS{Value: "user-123"},
								"schedule_name":       &dynamodbtypes.AttributeValueMemberS{Value: "test-schedule-1"},
								"schedule_expression": &dynamodbtypes.AttributeValueMemberS{Value: "cron(0 2 * * ? *)"},
								"schedule_type":       &dynamodbtypes.AttributeValueMemberS{Value: "recurring"},
								"status":              &dynamodbtypes.AttributeValueMemberS{Value: "active"},
							},
							{
								"schedule_id":         &dynamodbtypes.AttributeValueMemberS{Value: "sched-test-002"},
								"user_id":             &dynamodbtypes.AttributeValueMemberS{Value: "user-123"},
								"schedule_name":       &dynamodbtypes.AttributeValueMemberS{Value: "test-schedule-2"},
								"schedule_expression": &dynamodbtypes.AttributeValueMemberS{Value: "at(2026-01-22T15:00:00)"},
								"schedule_type":       &dynamodbtypes.AttributeValueMemberS{Value: "one-time"},
								"status":              &dynamodbtypes.AttributeValueMemberS{Value: "active"},
							},
						},
					}, nil
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:   "user with no schedules",
			userID: "user-empty",
			setupMock: func(ddb *mockDynamoDBClient) {
				ddb.queryFunc = func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{
						Items: []map[string]dynamodbtypes.AttributeValue{},
					}, nil
				}
			},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDDB := &mockDynamoDBClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockDDB)
			}

			client := &Client{
				dynamoClient: mockDDB,
				tableName:    "spawn-schedules",
			}

			schedules, err := client.ListSchedulesByUser(context.Background(), tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListSchedulesByUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(schedules) != tt.wantCount {
				t.Errorf("ListSchedulesByUser() returned %d schedules, want %d", len(schedules), tt.wantCount)
			}
		})
	}
}

func TestSaveSchedule(t *testing.T) {
	tests := []struct {
		name      string
		record    *ScheduleRecord
		setupMock func(*mockDynamoDBClient)
		wantErr   bool
	}{
		{
			name: "successful save",
			record: &ScheduleRecord{
				ScheduleID:         "sched-test-001",
				UserID:             "user-123",
				ScheduleName:       "test-schedule",
				ScheduleExpression: "cron(0 2 * * ? *)",
				ScheduleType:       "recurring",
				Status:             "active",
			},
			setupMock: func(ddb *mockDynamoDBClient) {
				ddb.putItemFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					if params.TableName == nil || *params.TableName != "spawn-schedules" {
						t.Errorf("Expected table name spawn-schedules, got %v", params.TableName)
					}
					return &dynamodb.PutItemOutput{}, nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDDB := &mockDynamoDBClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockDDB)
			}

			client := &Client{
				dynamoClient: mockDDB,
				tableName:    "spawn-schedules",
			}

			err := client.SaveSchedule(context.Background(), tt.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify TTL was set
			if tt.record.TTL == 0 {
				t.Error("SaveSchedule() did not set TTL")
			}
		})
	}
}

func TestGenerateScheduleID(t *testing.T) {
	id := GenerateScheduleID()
	if len(id) == 0 {
		t.Error("GenerateScheduleID() returned empty string")
	}
	if id[:6] != "sched-" {
		t.Errorf("GenerateScheduleID() should start with 'sched-', got %s", id)
	}
}

func TestGetExecutionHistory(t *testing.T) {
	tests := []struct {
		name       string
		scheduleID string
		limit      int
		setupMock  func(*mockDynamoDBClient)
		wantCount  int
		wantErr    bool
	}{
		{
			name:       "schedule with execution history",
			scheduleID: "sched-test-001",
			limit:      10,
			setupMock: func(ddb *mockDynamoDBClient) {
				ddb.queryFunc = func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{
						Items: []map[string]dynamodbtypes.AttributeValue{
							{
								"schedule_id": &dynamodbtypes.AttributeValueMemberS{Value: "sched-test-001"},
								"sweep_id":    &dynamodbtypes.AttributeValueMemberS{Value: "sweep-20260122-140530"},
								"status":      &dynamodbtypes.AttributeValueMemberS{Value: "success"},
							},
							{
								"schedule_id": &dynamodbtypes.AttributeValueMemberS{Value: "sched-test-001"},
								"sweep_id":    &dynamodbtypes.AttributeValueMemberS{Value: "sweep-20260123-140530"},
								"status":      &dynamodbtypes.AttributeValueMemberS{Value: "success"},
							},
						},
					}, nil
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:       "schedule with no history",
			scheduleID: "sched-empty",
			limit:      10,
			setupMock: func(ddb *mockDynamoDBClient) {
				ddb.queryFunc = func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{
						Items: []map[string]dynamodbtypes.AttributeValue{},
					}, nil
				}
			},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDDB := &mockDynamoDBClient{}
			if tt.setupMock != nil {
				tt.setupMock(mockDDB)
			}

			client := &Client{
				dynamoClient: mockDDB,
				tableName:    "spawn-schedules",
			}

			history, err := client.GetExecutionHistory(context.Background(), tt.scheduleID, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetExecutionHistory() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(history) != tt.wantCount {
				t.Errorf("GetExecutionHistory() returned %d items, want %d", len(history), tt.wantCount)
			}
		})
	}
}

package loyalty

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type OTPRepository struct {
	db *dynamodb.Client
}

func NewOTPRepository(db *dynamodb.Client) *OTPRepository {
	return &OTPRepository{db: db}
}

func (r *OTPRepository) Get(ctx context.Context, phone string) (*PhoneOTP, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableNamePhoneOTP),
		Key: map[string]types.AttributeValue{
			"phone": &types.AttributeValueMemberS{Value: phone},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Item) == 0 {
		return nil, nil
	}
	return decodePhoneOTP(out.Item), nil
}

func (r *OTPRepository) Put(ctx context.Context, row *PhoneOTP) error {
	if row == nil {
		return fmt.Errorf("otp row is nil")
	}
	item := map[string]types.AttributeValue{
		"phone":        &types.AttributeValueMemberS{Value: row.Phone},
		"otp_hash":     &types.AttributeValueMemberS{Value: row.OTPHash},
		"attempts":     &types.AttributeValueMemberN{Value: strconv.FormatInt(row.Attempts, 10)},
		"created_at":   &types.AttributeValueMemberN{Value: strconv.FormatInt(row.CreatedAt, 10)},
		"last_sent_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(row.LastSentAt, 10)},
		"ttl":          &types.AttributeValueMemberN{Value: strconv.FormatInt(row.TTL, 10)},
	}
	if row.PendingName != "" {
		item["pending_name"] = &types.AttributeValueMemberS{Value: row.PendingName}
	}
	_, err := r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableNamePhoneOTP),
		Item:      item,
	})
	return err
}

func (r *OTPRepository) Delete(ctx context.Context, phone string) error {
	_, err := r.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(TableNamePhoneOTP),
		Key: map[string]types.AttributeValue{
			"phone": &types.AttributeValueMemberS{Value: phone},
		},
	})
	return err
}

func (r *OTPRepository) IncrementAttempts(ctx context.Context, phone string) error {
	_, err := r.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(TableNamePhoneOTP),
		Key: map[string]types.AttributeValue{
			"phone": &types.AttributeValueMemberS{Value: phone},
		},
		UpdateExpression: aws.String("SET attempts = if_not_exists(attempts, :zero) + :one"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":zero": &types.AttributeValueMemberN{Value: "0"},
			":one":  &types.AttributeValueMemberN{Value: "1"},
		},
	})
	return err
}

func decodePhoneOTP(item map[string]types.AttributeValue) *PhoneOTP {
	row := &PhoneOTP{}
	if v, ok := item["phone"].(*types.AttributeValueMemberS); ok {
		row.Phone = v.Value
	}
	if v, ok := item["otp_hash"].(*types.AttributeValueMemberS); ok {
		row.OTPHash = v.Value
	}
	if v, ok := item["pending_name"].(*types.AttributeValueMemberS); ok {
		row.PendingName = v.Value
	}
	row.Attempts, _ = otpNumAttr(item, "attempts")
	row.CreatedAt, _ = otpNumAttr(item, "created_at")
	row.LastSentAt, _ = otpNumAttr(item, "last_sent_at")
	row.TTL, _ = otpNumAttr(item, "ttl")
	return row
}

func otpNumAttr(item map[string]types.AttributeValue, key string) (int64, error) {
	a, ok := item[key].(*types.AttributeValueMemberN)
	if !ok || a == nil {
		return 0, nil
	}
	return strconv.ParseInt(a.Value, 10, 64)
}

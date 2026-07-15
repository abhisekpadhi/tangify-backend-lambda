package billing

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// PointsWalletProvider reads and deducts loyalty points (implemented by loyalty package).
type PointsWalletProvider interface {
	GetPointsBalance(ctx context.Context, userID string) (int64, error)
}

// BillWithLineItemsRepository persists bill snapshots with embedded line items.
type BillWithLineItemsRepository struct {
	db        *dynamodb.Client
	tableName string
}

const DevTableNameBillsWithLineItems = "dev-tangify_bills_with_line_items"

func NewBillWithLineItemsRepository(
	db *dynamodb.Client,
	tableName ...string,
) *BillWithLineItemsRepository {
	name := TableNameBillsWithLineItems
	if len(tableName) > 0 && tableName[0] != "" {
		name = tableName[0]
	}
	return &BillWithLineItemsRepository{db: db, tableName: name}
}

func (r *BillWithLineItemsRepository) Get(ctx context.Context, id string) (*BillWithLineItems, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(r.tableName),
		ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Item) == 0 {
		return nil, nil
	}
	return decodeBillWithLineItems(out.Item)
}

func (r *BillWithLineItemsRepository) GetByStateKey(ctx context.Context, stateKey string) (*BillWithLineItems, error) {
	out, err := r.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String(GSIStateKey),
		KeyConditionExpression: aws.String("state_key = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sk": &types.AttributeValueMemberS{Value: stateKey},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, nil
	}
	return decodeBillWithLineItems(out.Items[0])
}

func (r *BillWithLineItemsRepository) Put(ctx context.Context, bill *BillWithLineItems) error {
	if bill == nil {
		return fmt.Errorf("bill is nil")
	}
	item, err := encodeBillWithLineItems(bill)
	if err != nil {
		return err
	}
	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	return err
}

// TransactCreate saves a new bill and optionally deducts loyalty points in one transaction.
func (r *BillWithLineItemsRepository) TransactCreate(
	ctx context.Context,
	bill *BillWithLineItems,
	walletUserID string,
	pointsToRedeem int64,
	walletTableName string,
	now int64,
) error {
	if bill == nil {
		return fmt.Errorf("bill is nil")
	}
	item, err := encodeBillWithLineItems(bill)
	if err != nil {
		return err
	}

	txItems := []types.TransactWriteItem{
		{
			Put: &types.Put{
				TableName:           aws.String(r.tableName),
				Item:                item,
				ConditionExpression: aws.String("attribute_not_exists(id)"),
			},
		},
	}

	if pointsToRedeem > 0 {
		if walletUserID == "" {
			return fmt.Errorf("wallet user id required when redeeming points")
		}
		txItems = append(txItems, types.TransactWriteItem{
			Update: &types.Update{
				TableName: aws.String(walletTableName),
				Key: map[string]types.AttributeValue{
					"user_id": &types.AttributeValueMemberS{Value: walletUserID},
				},
				UpdateExpression: aws.String(
					"SET points_balance = points_balance - :pts, lifetime_redeemed = if_not_exists(lifetime_redeemed, :zero) + :pts, updated_at = :now",
				),
				ConditionExpression: aws.String("points_balance >= :pts"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":pts":  &types.AttributeValueMemberN{Value: strconv.FormatInt(pointsToRedeem, 10)},
					":zero": &types.AttributeValueMemberN{Value: "0"},
					":now":  &types.AttributeValueMemberN{Value: strconv.FormatInt(now, 10)},
				},
			},
		})
	}

	_, err = r.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: txItems,
	})
	return err
}

func encodeBillWithLineItems(b *BillWithLineItems) (map[string]types.AttributeValue, error) {
	lineItems := make([]types.AttributeValue, 0, len(b.LineItems))
	for _, li := range b.LineItems {
		lineItems = append(lineItems, &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"name":     &types.AttributeValueMemberS{Value: li.Name},
			"quantity": &types.AttributeValueMemberN{Value: strconv.Itoa(li.Quantity)},
			"price":    &types.AttributeValueMemberN{Value: strconv.FormatInt(li.Price, 10)},
		}})
	}

	discounts := make([]types.AttributeValue, 0, len(b.Discounts))
	for _, d := range b.Discounts {
		discounts = append(discounts, &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"id":          &types.AttributeValueMemberS{Value: d.ID},
			"type":        &types.AttributeValueMemberS{Value: d.Type},
			"amount":      &types.AttributeValueMemberN{Value: strconv.FormatInt(d.Amount, 10)},
			"description": &types.AttributeValueMemberS{Value: d.Description},
		}})
	}

	taxes := make([]types.AttributeValue, 0, len(b.Taxes))
	for _, t := range b.Taxes {
		taxes = append(taxes, &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"id":              &types.AttributeValueMemberS{Value: t.ID},
			"name":            &types.AttributeValueMemberS{Value: t.Name},
			"rate_in_bps":     &types.AttributeValueMemberN{Value: strconv.Itoa(t.RateInBps)},
			"amount_in_paise": &types.AttributeValueMemberN{Value: strconv.FormatInt(t.AmountInPaise, 10)},
		}})
	}

	tids := make([]types.AttributeValue, 0, len(b.TableIDs))
	for _, t := range b.TableIDs {
		tids = append(tids, &types.AttributeValueMemberS{Value: t})
	}

	m := map[string]types.AttributeValue{
		"id":                      &types.AttributeValueMemberS{Value: b.ID},
		"session_id":              &types.AttributeValueMemberS{Value: b.SessionID},
		"staff_id":                &types.AttributeValueMemberS{Value: b.StaffID},
		"customer_id":             &types.AttributeValueMemberS{Value: b.CustomerID},
		"payment_method":          &types.AttributeValueMemberS{Value: b.PaymentMethod},
		"payment_status":          &types.AttributeValueMemberS{Value: b.PaymentStatus},
		"created_at":              &types.AttributeValueMemberN{Value: strconv.FormatInt(b.CreatedAt, 10)},
		"updated_at":              &types.AttributeValueMemberN{Value: strconv.FormatInt(b.UpdatedAt, 10)},
		"line_items":              &types.AttributeValueMemberL{Value: lineItems},
		"discounts":               &types.AttributeValueMemberL{Value: discounts},
		"taxes":                   &types.AttributeValueMemberL{Value: taxes},
		"table_ids":               &types.AttributeValueMemberL{Value: tids},
		"total_tax_in_paise":      &types.AttributeValueMemberN{Value: strconv.FormatInt(b.TotalTaxInPaise, 10)},
		"total_discount_in_paise": &types.AttributeValueMemberN{Value: strconv.FormatInt(b.TotalDiscountInPaise, 10)},
		"total_amount_in_paise":   &types.AttributeValueMemberN{Value: strconv.FormatInt(b.TotalAmountInPaise, 10)},
	}
	if b.StateKey != "" {
		m["state_key"] = &types.AttributeValueMemberS{Value: b.StateKey}
	}
	return m, nil
}

func decodeBillWithLineItems(item map[string]types.AttributeValue) (*BillWithLineItems, error) {
	b := &BillWithLineItems{}
	if v, ok := item["id"].(*types.AttributeValueMemberS); ok {
		b.ID = v.Value
	}
	if v, ok := item["state_key"].(*types.AttributeValueMemberS); ok {
		b.StateKey = v.Value
	}
	if v, ok := item["session_id"].(*types.AttributeValueMemberS); ok {
		b.SessionID = v.Value
	}
	if v, ok := item["staff_id"].(*types.AttributeValueMemberS); ok {
		b.StaffID = v.Value
	}
	if v, ok := item["customer_id"].(*types.AttributeValueMemberS); ok {
		b.CustomerID = v.Value
	}
	if v, ok := item["payment_method"].(*types.AttributeValueMemberS); ok {
		b.PaymentMethod = v.Value
	}
	if v, ok := item["payment_status"].(*types.AttributeValueMemberS); ok {
		b.PaymentStatus = v.Value
	}
	b.CreatedAt, _ = numAttr(item, "created_at")
	b.UpdatedAt, _ = numAttr(item, "updated_at")
	b.TotalTaxInPaise, _ = numAttr(item, "total_tax_in_paise")
	b.TotalDiscountInPaise, _ = numAttr(item, "total_discount_in_paise")
	b.TotalAmountInPaise, _ = numAttr(item, "total_amount_in_paise")

	if l, ok := item["table_ids"].(*types.AttributeValueMemberL); ok {
		for _, e := range l.Value {
			if sv, ok := e.(*types.AttributeValueMemberS); ok {
				b.TableIDs = append(b.TableIDs, sv.Value)
			}
		}
	}
	if l, ok := item["line_items"].(*types.AttributeValueMemberL); ok {
		for _, e := range l.Value {
			m, ok := e.(*types.AttributeValueMemberM)
			if !ok {
				continue
			}
			li := LineItemV0{}
			if s, ok := m.Value["name"].(*types.AttributeValueMemberS); ok {
				li.Name = s.Value
			}
			qty, _ := atoiAttr(m.Value, "quantity")
			li.Quantity = qty
			li.Price, _ = numAttr(m.Value, "price")
			b.LineItems = append(b.LineItems, li)
		}
	}
	if l, ok := item["discounts"].(*types.AttributeValueMemberL); ok {
		for _, e := range l.Value {
			dm, ok := e.(*types.AttributeValueMemberM)
			if !ok {
				continue
			}
			d := DiscountType{}
			if s, ok := dm.Value["id"].(*types.AttributeValueMemberS); ok {
				d.ID = s.Value
			}
			if s, ok := dm.Value["type"].(*types.AttributeValueMemberS); ok {
				d.Type = s.Value
			}
			d.Amount, _ = numAttr(dm.Value, "amount")
			if s, ok := dm.Value["description"].(*types.AttributeValueMemberS); ok {
				d.Description = s.Value
			}
			b.Discounts = append(b.Discounts, d)
		}
	}
	if l, ok := item["taxes"].(*types.AttributeValueMemberL); ok {
		for _, e := range l.Value {
			tm, ok := e.(*types.AttributeValueMemberM)
			if !ok {
				continue
			}
			t := TaxType{}
			if s, ok := tm.Value["id"].(*types.AttributeValueMemberS); ok {
				t.ID = s.Value
			}
			if s, ok := tm.Value["name"].(*types.AttributeValueMemberS); ok {
				t.Name = s.Value
			}
			t.RateInBps, _ = atoiAttr(tm.Value, "rate_in_bps")
			t.AmountInPaise, _ = numAttr(tm.Value, "amount_in_paise")
			b.Taxes = append(b.Taxes, t)
		}
	}
	return b, nil
}

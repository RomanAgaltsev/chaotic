// Command aws-dynamodb-retry demonstrates that the AWS SDK's own retryer recovers
// from a transient outage injected by the chaotic adapter/aws middleware — with no
// AWS account and no Docker, using a local httptest server as the DynamoDB endpoint.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// GetItem fetches a key; with the chaos middleware installed and the SDK's default
// retryer, a transient outage is absorbed by the SDK's retries.
func GetItem(ctx context.Context, ddb *dynamodb.Client, table, id string) (*dynamodb.GetItemOutput, error) {
	return ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			id: &types.AttributeValueMemberS{Value: id},
		},
	})
}

func main() {
	fmt.Fprintln(os.Stdout, "run `go test` in this directory to see the SDK retryer survive a chaos outage")
}

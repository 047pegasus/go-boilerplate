package custom_utils

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"
)

func PingValkeyBuildAndExecute(ctx context.Context, vc valkey.Client) (string, error) {
	pingCmd := vc.B().Ping().Build()
	result, err := vc.Do(ctx, pingCmd).ToString()
	if err != nil {
		return "", fmt.Errorf("failed to ping valkey: %w", err)
	}
	return result, nil
}

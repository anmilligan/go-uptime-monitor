package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// AlertDeduplicator provides temporary Redis-based deduplication
// for alert processing across application instances.
type AlertDeduplicator struct {
	client *goredis.Client
}

func NewAlertDeduplicator(client *goredis.Client) *AlertDeduplicator {
	return &AlertDeduplicator{
		client: client,
	}
}

// Claim attempts to reserve alert processing for the specified incident
// and alert type. It returns false if the claim already exists.
func (d *AlertDeduplicator) Claim(ctx context.Context, incidentID int, alertType string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, fmt.Errorf(
			"alert deduplication ttl must be positive",
		)
	}

	key := alertDedupKey(
		incidentID,
		alertType,
	)

	claimed, err := d.client.SetNX(
		ctx,
		key,
		"1",
		ttl,
	).Result()

	if err != nil {
		return false, err
	}

	return claimed, nil
}

// Release removes an existing alert processing claim.
func (d *AlertDeduplicator) Release(ctx context.Context, incidentID int, alertType string) error {
	key := alertDedupKey(
		incidentID,
		alertType,
	)

	return d.client.Del(
		ctx,
		key,
	).Err()
}

// alertDedupKey builds the Redis key used to deduplicate an alert.
func alertDedupKey(incidentID int, alertType string) string {
	return fmt.Sprintf(
		"uptime:alert:%d:%s",
		incidentID,
		strings.ToLower(alertType),
	)
}

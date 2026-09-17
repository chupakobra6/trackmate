package postgres

import (
	"context"
	"time"
)

func (q *Queries) DeliveryReady(ctx context.Context, operation string, id int64, now time.Time) (bool, error) {
	var ready bool
	err := q.db.QueryRow(ctx, `SELECT NOT EXISTS (SELECT 1 FROM worker_delivery_retries WHERE operation=$1 AND entity_id=$2 AND (exhausted OR next_attempt_at>$3))`, operation, id, now).Scan(&ready)
	return ready, err
}

func (q *Queries) RecordDeliveryFailure(ctx context.Context, operation string, id int64, now time.Time, cause string, suspend bool) error {
	_, err := q.db.Exec(ctx, `INSERT INTO worker_delivery_retries(operation,entity_id,attempts,next_attempt_at,last_error,updated_at,exhausted)
 VALUES($1,$2,1,$3::timestamptz+interval '1 minute',$4,$3,$5)
 ON CONFLICT(operation,entity_id) DO UPDATE SET
 attempts=LEAST(worker_delivery_retries.attempts+1,8),
 next_attempt_at=$3::timestamptz + interval '1 minute' * power(2,LEAST(worker_delivery_retries.attempts,6)),
 exhausted=$5 OR worker_delivery_retries.attempts+1>=8,
 last_error=$4,updated_at=$3`, operation, id, now, cause, suspend)
	return err
}

func (q *Queries) ClearDeliveryRetry(ctx context.Context, operation string, id int64) error {
	_, err := q.db.Exec(ctx, `DELETE FROM worker_delivery_retries WHERE operation=$1 AND entity_id=$2`, operation, id)
	return err
}

func (q *Queries) DeliveryRetryIDs(ctx context.Context, operation string, now time.Time) ([]int64, error) {
	rows, err := q.db.Query(ctx, `SELECT entity_id FROM worker_delivery_retries WHERE operation=$1 AND NOT exhausted AND next_attempt_at<=$2 ORDER BY entity_id`, operation, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

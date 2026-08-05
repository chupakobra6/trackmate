package delivery

import (
	"context"
	"time"

	"github.com/igor/trackmate/internal/telegram"
)

const compensationTimeout = 5 * time.Second

func CompensateSentMessage(tg telegram.API, chatID int64, messageID int64, afterDelete func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), compensationTimeout)
	defer cancel()
	if err := tg.DeleteMessage(ctx, chatID, messageID); err != nil {
		return err
	}
	if afterDelete != nil {
		return afterDelete(ctx)
	}
	return nil
}

package templates

import (
	"context"

	"vekr/internal/i18n"
)

func t(ctx context.Context, id string) string {
	return i18n.T(ctx, id)
}

func tData(ctx context.Context, id string, data map[string]any) string {
	return i18n.TData(ctx, id, data)
}

func tCount(ctx context.Context, id string, count int, data map[string]any) string {
	return i18n.TCount(ctx, id, count, data)
}

// Package grpcpaging provides helpers for collecting paginated gRPC results.
package grpcpaging

import (
	"context"
	"strings"
)

// CollectPages fetches all pages from a paginated gRPC endpoint, mapping each
// item through mapItem. The mapItem function returns (mapped, ok) — items
// where ok is false are skipped.
func CollectPages[T any, R any](
	ctx context.Context,
	pageSize int32,
	fetch func(ctx context.Context, pageToken string) (items []R, nextToken string, err error),
	mapItem func(R) (T, bool),
) ([]T, error) {
	return collectPages[T, R](ctx, pageSize, 0, fetch, mapItem)
}

// CollectPagesMax is like CollectPages but stops after maxPages iterations.
// Use this to bound unbounded pagination (e.g., notification inbox).
func CollectPagesMax[T any, R any](
	ctx context.Context,
	pageSize int32,
	maxPages int,
	fetch func(ctx context.Context, pageToken string) (items []R, nextToken string, err error),
	mapItem func(R) (T, bool),
) ([]T, error) {
	if maxPages <= 0 {
		maxPages = 1
	}
	return collectPages[T, R](ctx, pageSize, maxPages, fetch, mapItem)
}

func collectPages[T any, R any](
	ctx context.Context,
	pageSize int32,
	maxPages int,
	fetch func(ctx context.Context, pageToken string) (items []R, nextToken string, err error),
	mapItem func(R) (T, bool),
) ([]T, error) {
	result := make([]T, 0, pageSize)
	pageToken := ""
	for page := 0; maxPages == 0 || page < maxPages; page++ {
		items, nextToken, err := fetch(ctx, pageToken)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if mapped, ok := mapItem(item); ok {
				result = append(result, mapped)
			}
		}
		nextToken = strings.TrimSpace(nextToken)
		if nextToken == "" || nextToken == pageToken {
			break
		}
		pageToken = nextToken
	}
	return result, nil
}

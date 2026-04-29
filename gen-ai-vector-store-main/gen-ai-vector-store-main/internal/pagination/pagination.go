/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package pagination

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/url"
	"strconv"
)

type Pagination struct {
	NextCursor string `json:"nextCursor,omitempty"` // NextCursor is the cursor to fetch the next page of data
	Limit      int    `json:"limit,omitempty"`      // Current limit of items per page (current page)
	ItemsTotal int    `json:"itemsTotal,omitempty"` // Total number of items in the collection
	ItemsLeft  int    `json:"itemsLeft,omitempty"`  // Number of items left in the collection after the current page
}

func GetPaginationParameters(c *gin.Context) (cursor string, limit int, err error) {
	allRequested := c.DefaultQuery("all", "false") == "true" // Fetch all data if requested
	if allRequested {
		return "", 0, nil
	}
	cursor, err = url.PathUnescape(c.DefaultQuery("cursor", "")) // Decode the cursor
	if err != nil {
		return "", 0, fmt.Errorf("error decoding cursor: %w", err)
	}
	limit, err = strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(PaginationDefaultLimit)))
	if err != nil {
		return "", 0, fmt.Errorf("invalid limit: %w", err)
	}
	if limit > PaginationMaxLimit {
		return "", 0, fmt.Errorf("max allowed limit is %d", PaginationMaxLimit)
	}
	return cursor, limit, nil
}

// CalculatePagination returns a pagination object for any list of items
// T is the type of items in the list
// getID is a function that extracts the ID from an item to use as cursor
func CalculatePagination[T any](items []T, limit int, itemsTotal, itemsLeft int, getID func(T) string) (p Pagination) {
	if limit <= 0 {
		return p
	}
	p.Limit = limit
	p.ItemsLeft = itemsLeft
	p.ItemsTotal = itemsTotal
	if itemsLeft > 0 {
		// If there are more items after the current page, set the next cursor
		lastItemID := getID(items[len(items)-1])
		p.NextCursor = url.PathEscape(lastItemID)
	}
	return p
}

const PaginationDefaultLimit = 500

const PaginationMaxLimit = 10000

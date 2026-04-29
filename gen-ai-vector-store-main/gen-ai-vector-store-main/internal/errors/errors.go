/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package errors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var logger = log.GetNamedLogger("errors")

var (
	ErrorParsingTheRequest       = fmt.Errorf("error while parsing the request")
	ErrorProcessingTheRequest    = fmt.Errorf("error while processing the request")
	ErrorUnmarshallingTheRequest = fmt.Errorf("error while unmarshalling the request")
)

// IsTimeout checks if the error is a timeout error
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	errMsg := strings.ToLower(err.Error())
	timeoutIndicators := []string{
		"context deadline exceeded",
		"client.timeout exceeded",
		"timeout",
		"timed out",
		"deadline exceeded",
	}
	for _, indicator := range timeoutIndicators {
		if strings.Contains(errMsg, indicator) {
			return true
		}
	}
	return false
}

var (
	ErrorDeletingInProgressDocsPattern = regexp.MustCompile("cannot delete IN_PROGRESS documents")
	ErrorProcessingRequestPattern      = regexp.MustCompile("error while processing the request")
	ErrorParsingRequestPattern         = regexp.MustCompile("error while parsing the request")
	ErrorUnmarshallingPattern          = regexp.MustCompile("error while unmarshalling the request")
	ErrorInvalidRequestPattern         = regexp.MustCompile("invalid request")
)

// ResponseError is a wrapper for returning more sophisticated errors from endpoints
type ResponseError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
}

// Error is a string representation of ResponseError
func (a ResponseError) Error() string {
	return fmt.Sprintf("%d: %s", a.Code, a.Message)
}

// ToResponseError is a helper function to convert the error to the ResponseError - type which
// unifies the API error responses format
func ToResponseError(err error) ResponseError {
	var rErr ResponseError
	var pgErr *pgconn.PgError
	if err == nil {
		err = fmt.Errorf("")
	}
	if ok := errors.As(err, &pgErr); ok && pgErr.Code == pgerrcode.UndefinedTable {
		rErr = ResponseError{
			Message: "isolationID or collectionName does not match any of existing resources",
			Code:    http.StatusBadRequest,
		}

	} else if ok = errors.As(err, &rErr); !ok {
		rErr = ResponseError{
			Code:    mapErrorToHTTPCode(err),
			Message: err.Error(),
		}
	}
	return rErr
}

func mapErrorToHTTPCode(err error) int {
	logger.Debug(fmt.Sprintf("mapErrorToHTTPCode: err.Error() = %s", err.Error()))
	if ErrorDeletingInProgressDocsPattern.MatchString(err.Error()) {
		return http.StatusConflict
	} else if ErrorProcessingRequestPattern.MatchString(err.Error()) {
		return http.StatusBadRequest
	} else if ErrorParsingRequestPattern.MatchString(err.Error()) {
		return http.StatusBadRequest
	} else if ErrorUnmarshallingPattern.MatchString(err.Error()) {
		return http.StatusBadRequest
	} else if ErrorInvalidRequestPattern.MatchString(err.Error()) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

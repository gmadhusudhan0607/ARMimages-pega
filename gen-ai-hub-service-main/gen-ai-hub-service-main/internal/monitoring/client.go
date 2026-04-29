/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/repository"
)

func publishEvent(ctx context.Context, evt *repository.Event) {
	l := cntx.LoggerFromContext(ctx)

	uri := os.Getenv("MONITORING_ENDPOINT")
	jsonStr, err := json.Marshal(evt)
	if err != nil {
		l.Sugar().Error(err)
		return
	}
	resp, err := http.Post(uri, "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		l.Sugar().Error("unable to monitor request - error publishing Event")
		return
	}
	defer resp.Body.Close()
	// Drain the body to allow the underlying TCP connection to be reused
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		l.Sugar().Errorf("unable to monitor request - error calling %s with code %d", uri, resp.StatusCode)
		return
	}
}

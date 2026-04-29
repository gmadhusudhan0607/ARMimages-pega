/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package cntxtest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx/cntxtest"
)

func TestNewContext(t *testing.T) {
	ctx := cntxtest.NewContext("test")
	assert.NotNil(t, ctx)

	// Verify default values are set correctly
	assert.True(t, cntx.IsInfinityPlatform(ctx))
	assert.False(t, cntx.IsUseSax(ctx))
	assert.False(t, cntx.IsUseGenAiInfraModels(ctx))
}

func TestWithSaxConfigPath(t *testing.T) {
	ctx := cntxtest.NewContext("test")
	ctx = cntxtest.WithSaxConfigPath(ctx, "/custom/path")

	// Verify custom path is set
	assert.Equal(t, "/custom/path", cntx.GetSaxConfigPath(ctx))
}

func TestWithUseGenAIInfra(t *testing.T) {
	ctx := cntxtest.NewContext("test")
	assert.False(t, cntx.IsUseGenAiInfraModels(ctx))

	ctx = cntxtest.WithUseGenAIInfra(ctx, true)
	assert.True(t, cntx.IsUseGenAiInfraModels(ctx))
}

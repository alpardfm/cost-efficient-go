//go:build tools
// +build tools

// Package tools tracks tool and library dependencies that are used by patterns
// but not yet imported in the main module. This file ensures go mod tidy
// retains these dependencies in go.mod.
package tools

import (
	_ "github.com/leanovate/gopter"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/rs/zerolog"
	_ "go.uber.org/zap"
)

/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package helpers

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

const (
	defaultIsolationAutoCreationMaxStorageSize = "1GB"
)

var _isolationAutoCreation *bool
var _isolationAutoCreationMaxStorageSize *string

func GetEnvOrPanic(name string) string {
	val, present := os.LookupEnv(name)
	if !present {
		panic(fmt.Sprintf("Env variable '%s' is required", name))
	}
	return val
}

func GetEnvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func GetEnvOrDefaultInt64(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	valueInt, err := strconv.ParseInt(value, 10, 0)
	if err != nil {
		return fallback
	}
	return valueInt
}

func IsDBLocal() bool              { return os.Getenv("DB_LOCAL") == "true" }
func UseLegacyAttributesIDs() bool { return os.Getenv("DB_USE_LEGACY_ATTRIBUTE_IDS") == "true" }
func IsSaxDisabled() bool          { return os.Getenv("SAX_DISABLED") == "true" }
func IsSaxClientDisabled() bool    { return os.Getenv("SAX_CLIENT_DISABLED") == "true" }
func IsIsolationIDVerificationDisabled() bool {
	return os.Getenv("ISOLATION_ID_VERIFICATION_DISABLED") == "true" || IsSaxDisabled()
}
func IsLogPerformanceTrace() bool { return os.Getenv("LOG_PERFORMANCE_TRACE") == "true" }

func IsReadOnlyMode() bool        { return os.Getenv("READ_ONLY_MODE") == "true" }
func IsTroubleshootingMode() bool { return os.Getenv("TROUBLESHOOTING_MODE") == "true" }
func IsEncourageSemSearchIndexUseEnabled() bool {
	return os.Getenv("ENCOURAGE_SEM_SEARCH_INDEX_USE") == "true"
}
func IsRuntimeConfigurationViaHeadersEnabled() bool {
	return GetEnvOrDefault("ENABLE_RUNTIME_HEADER_CONFIG", "false") == "true"
}

func IsEmulationEnabled() bool {
	return os.Getenv("EMULATION_MODE") == "true"
}

func GetEmulationMinTime() int64 {
	return GetEnvOrDefaultInt64("EMULATION_MIN_TIME", 100)
}

func GetEmulationMaxTime() int64 {
	return GetEnvOrDefaultInt64("EMULATION_MAX_TIME", 1000)
}
func GetForcedMinRequiredSchemaVersion() string {
	return GetEnvOrDefault("DB_SCHEMA_FORCED_MIN_REQUIRED_VERSION", "")
}

func IsIsolationAutoCreationEnabled() bool {
	if _isolationAutoCreation == nil {
		iac := GetEnvOrDefault("ISOLATION_AUTO_CREATION", "false") == "true"
		_isolationAutoCreation = &iac
	}
	return *_isolationAutoCreation
}

// IsSaxTokenCacheEnabled returns true if JWT token caching is enabled
func IsSaxTokenCacheEnabled() bool {
	return GetEnvOrDefault("SAX_TOKEN_CACHE_ENABLED", "true") == "true"
}

func GetIsolationAutoCreationMaxStorageSize() string {
	if _isolationAutoCreationMaxStorageSize == nil {
		maxStorageSize := GetEnvOrDefault("ISOLATION_AUTO_CREATION_MAX_STORAGE_SIZE", defaultIsolationAutoCreationMaxStorageSize)
		_isolationAutoCreationMaxStorageSize = &maxStorageSize
	}
	return *_isolationAutoCreationMaxStorageSize
}

// LogTruncated truncate number of printed elements to 'count'
// To avoid ERROR: bufio.Scanner: token too long ( > 1Mb )
func LogTruncated(l *zap.Logger, message string, count int, data interface{}) {
	s := reflect.ValueOf(data)
	if s.Len() > count {
		l.Debug("output truncated",
			zap.String("message", message),
			zap.Any("data", s.Slice(0, count).Interface()),
			zap.Int("truncated_count", count),
			zap.Int("total_count", s.Len()),
		)
	} else {
		l.Debug("output",
			zap.String("message", message),
			zap.Any("data", data),
		)
	}
}

// Differences in floating-point behavior are particularly noticeable on different architectures (e.g., x86 vs. ARM) or even different CPU models within the same architecture.
// This can lead to subtle differences in the results of floating-point calculations, especially when comparing results across different platforms.
// Docker containers share the host kernel, and certain operations may depend on system-level floating-point settings or implementations.
// Different Linux distributions (or even different kernel versions) can have minor variations in floating-point handling and optimization settings, potentially impacting calculations
// It impacts pgvector cosine distance calculation which is based on floating-point operations. So that we cut off the floating-point precision to 4 digits after dot for Integration tests.
// The default value is 0, which means no cut off. The value can be set by the environment variable PGVECTOR_DISTANCE_PRECISION.
var _defaultDistancePrecision = "0"
var _distancePrecision = -1

func IsCutOffDistancePrecisionEnabled() bool {
	if _distancePrecision == -1 {
		var err error
		prStr := GetEnvOrDefault("PGVECTOR_DISTANCE_PRECISION", _defaultDistancePrecision)
		_distancePrecision, err = strconv.Atoi(prStr)
		if err != nil {
			panic(fmt.Sprintf("failed to parse PGVECTOR_DISTANCE_PRECISION: %s", err.Error()))
		}
	}
	return _distancePrecision > 0
}

func CutOffDistancePrecision(distance float64) float64 {
	if _distancePrecision > 0 {
		return CutOffFloatingPointPrecision(distance, _distancePrecision)
	}
	return distance
}

func CutOffFloatingPointPrecision(value float64, numbersAfterDot int) float64 {
	multiplier := 1.0
	for i := 0; i < numbersAfterDot; i++ {
		multiplier *= 10
	}
	return float64(int(value*multiplier)) / multiplier
}

func SplitTableName(tableName string) (string, string) {
	path := strings.Split(tableName, ".")
	if len(path) == 2 {
		return path[0], path[1]
	}
	return "public", tableName
}

var truncateMaxLength = int(GetEnvOrDefaultInt64("TRUNCATE_MAX_LENGTH", 1024))

func ToTruncatedString(value interface{}) string {
	// if value is []byte , convert it to string
	var valueStr string
	if v, ok := value.([]byte); ok {
		valueStr = string(v)
	} else {
		valueStr = fmt.Sprintf("%#v", value)
	}
	if len(valueStr) > truncateMaxLength {
		return valueStr[:truncateMaxLength] + "..."
	}
	return valueStr
}

func IsValidHeaderName(name string) bool {
	// Header names should only contain ASCII visible characters and no whitespace
	for _, c := range name {
		if c <= 32 || c >= 127 || c == ':' {
			return false
		}
	}
	return name != ""
}

func SanitizeHeaderValue(value string) string {
	// CWE-113 Replace CR, LF and other control characters
	var result []rune
	for _, c := range value {
		if c == '\r' || c == '\n' || c < 32 {
			// Skip control characters
			continue
		}
		result = append(result, c)
	}
	return string(result)
}

// ParseIntOrDefault parses a string to int, returning default value on error
func ParseIntOrDefault(value string, defaultValue int) int {
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return defaultValue
}

// Usage metrics configuration functions
func IsUsageMetricsEnabled() bool {
	return GetEnvOrDefault("USAGE_METRICS_ENABLED", "false") == "true"
}

func GetUsageMetricsUploadIntervalSeconds() int {
	return ParseIntOrDefault(GetEnvOrDefault("USAGE_METRICS_UPLOAD_INTERVAL_SECONDS", "3600"), 3600)
}

func GetUsageMetricsMaxPayloadSizeBytes() int {
	return ParseIntOrDefault(GetEnvOrDefault("USAGE_METRICS_MAX_PAYLOAD_SIZE", "819200"), 819200)
}

func GetUsageMetricsRetryCount() int {
	return ParseIntOrDefault(GetEnvOrDefault("USAGE_METRICS_RETRY_COUNT", "3"), 3)
}

func GetUsageMetricsRequestTimeoutSeconds() int {
	return ParseIntOrDefault(GetEnvOrDefault("USAGE_METRICS_REQUEST_TIMEOUT_SECS", "30"), 30)
}

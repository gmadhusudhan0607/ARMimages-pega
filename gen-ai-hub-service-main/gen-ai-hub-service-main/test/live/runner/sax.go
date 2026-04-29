/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// SaxConfig holds SAX token issuing configuration.
type SaxConfig struct {
	Profile  string
	SecretID string
	Region   string
}

// saxProfiles lists the AWS profiles to try in order when making AWS API calls.
// "dev-ai" is the primary SSO profile; "default" is the fallback for environments
// where dev-ai is not configured or its session has expired.
var saxProfiles = []string{"dev-ai", "default"}

// resolvedProfile caches the first profile that successfully authenticates.
var (
	resolvedProfile     string
	resolvedProfileOnce sync.Once
)

// resolveSaxProfile probes each candidate profile with an STS GetCallerIdentity
// call and returns the first one that succeeds. The result is cached for the
// process lifetime.
func resolveSaxProfile(region string) string {
	resolvedProfileOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, profile := range saxProfiles {
			cfg, err := config.LoadDefaultConfig(ctx,
				config.WithSharedConfigProfile(profile),
				config.WithRegion(region),
			)
			if err != nil {
				logVerbosef("  AWS profile %q: failed to load config: %v\n", profile, err)
				continue
			}
			client := sts.NewFromConfig(cfg)
			result, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
			if err != nil {
				logVerbosef("  AWS profile %q: credentials not valid: %v\n", profile, err)
				continue
			}
			logVerbosef("  AWS profile %q: authenticated as %s\n", profile, aws.ToString(result.Arn))
			resolvedProfile = profile
			return
		}
		// No profile worked — fall back to dev-ai so error messages stay familiar.
		logVerbose("  Warning: no AWS profile succeeded, falling back to dev-ai")
		resolvedProfile = saxProfiles[0]
	})
	return resolvedProfile
}

// saxCellRegions maps SAX cell names to their AWS regions.
var saxCellRegions = map[string]string{
	"us": "us-east-1",
	"eu": "eu-central-1",
}

// SaxConfigForCell returns a SaxConfig for the given SAX cell.
// Falls back to us-east-1 for unknown cells. The AWS profile is resolved
// dynamically by probing candidate profiles (dev-ai, then default).
func SaxConfigForCell(cell string) SaxConfig {
	region, ok := saxCellRegions[cell]
	if !ok {
		region = "us-east-1"
	}
	return SaxConfig{Profile: resolveSaxProfile(region), Region: region}
}

// saxBackingServicePattern matches SAX backing-service secret names (prefix + UUID start).
var saxBackingServicePattern = regexp.MustCompile(`^sax/backing-services/[0-9a-f]{8}`)

// jwtExpiryMargin is subtracted from the JWT exp claim when checking freshness.
const jwtExpiryMargin = 1 * time.Minute

// fileCache is a process-safe file-backed cache for a single string value.
// It uses flock for cross-process mutual exclusion and atomic rename for
// safe writes.
type fileCache struct {
	path     string
	lockPath string
}

// saxCacheDir is a fixed shared directory for cache files, avoiding os.TempDir()
// which on macOS returns a per-user per-boot /var/folders path that may differ
// across terminal sessions.
const saxCacheDir = "/tmp"

// newFileCache creates a fileCache with the given filename in saxCacheDir.
func newFileCache(name string) *fileCache {
	path := filepath.Join(saxCacheDir, name)
	return &fileCache{path: path, lockPath: path + ".lock"}
}

// arnCacheForCell returns the per-cell file cache for the SAX secret ARN.
func arnCacheForCell(cell string) *fileCache {
	return newFileCache(fmt.Sprintf("sax-secret-arn-%s.cache", cell))
}

// jwtCacheForCell returns the per-cell file cache for the JWT token.
func jwtCacheForCell(cell string) *fileCache {
	return newFileCache(fmt.Sprintf("sax-jwt-token-%s.cache", cell))
}

// saxConfigCacheForCell returns the per-cell file cache for the SAX client config JSON.
func saxConfigCacheForCell(cell string) *fileCache {
	return newFileCache(fmt.Sprintf("sax-config-%s.cache", cell))
}

// jwtExpTime extracts the exp claim from a JWT token string.
// Returns zero time on any parse error.
func jwtExpTime(token string) time.Time {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) < 2 {
		return time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp json.Number `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}
	}
	exp, err := claims.Exp.Int64()
	if err != nil {
		return time.Time{}
	}
	return time.Unix(exp, 0)
}

// acquireLock opens the lock file and acquires an exclusive flock.
// The caller must call the returned unlock function when done.
func (c *fileCache) acquireLock() (unlock func(), err error) {
	f, err := os.OpenFile(c.lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("flock: %w", err)
	}
	return func() {
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
			fmt.Printf("Warning: failed to unlock %s: %v\n", c.lockPath, err)
		}
		if err := f.Close(); err != nil {
			fmt.Printf("Warning: failed to close lock file %s: %v\n", c.lockPath, err)
		}
	}, nil
}

// read returns the cached value, or empty string if not cached.
// Must be called while holding the lock.
func (c *fileCache) read() string {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// write atomically writes a value to the cache file.
// Uses write-to-temp-then-rename so concurrent readers never see a partial value.
// Must be called while holding the lock.
func (c *fileCache) write(value string) error {
	tmp, err := os.CreateTemp(saxCacheDir, filepath.Base(c.path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(value); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, c.path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// invalidate acquires the lock and removes the cached value.
func (c *fileCache) invalidate() {
	unlock, err := c.acquireLock()
	if err != nil {
		fmt.Printf("Warning: failed to acquire lock for cache invalidation: %v\n", err)
		return
	}
	defer unlock()
	if err := os.Remove(c.path); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to remove cache file %s: %v\n", c.path, err)
	}
}

// resolveSecretARN returns the SAX secret ARN for the given cell. It acquires
// an exclusive file lock, checks the per-cell cache, and falls back to AWS
// Secrets Manager discovery on cache miss.
func resolveSecretARN(cell, profile, region string) (string, error) {
	cache := arnCacheForCell(cell)
	unlock, err := cache.acquireLock()
	if err != nil {
		return "", fmt.Errorf("acquire cache lock: %w", err)
	}
	defer unlock()

	if arn := cache.read(); arn != "" {
		logVerbosef("  Using cached secret ARN from %s: %s\n", cache.path, arn)
		return arn, nil
	}

	arn, err := discoverSecretARN(profile, region)
	if err != nil {
		return "", err
	}
	if err := cache.write(arn); err != nil {
		fmt.Printf("Warning: failed to cache secret ARN to %s: %v\n", cache.path, err)
	}
	logVerbosef("  Discovered and cached secret ARN to %s: %s\n", cache.path, arn)
	return arn, nil
}

// discoverSecretARN dynamically discovers the first SAX backing-service secret ARN
// by querying AWS Secrets Manager using the AWS SDK.
func discoverSecretARN(profile, region string) (string, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
		config.WithRegion(region),
	)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config (profile=%s, region=%s): %w", profile, region, err)
	}

	client := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.ListSecretsInput{
		Filters: []types.Filter{
			{
				Key:    "name",
				Values: []string{"sax/backing-services/"},
			},
		},
	}

	var nextToken *string
	for {
		input.NextToken = nextToken
		result, err := client.ListSecrets(ctx, input)
		if err != nil {
			return "", fmt.Errorf("ListSecrets failed: %w", err)
		}
		for _, secret := range result.SecretList {
			if secret.Name != nil && saxBackingServicePattern.MatchString(*secret.Name) {
				if secret.ARN == nil {
					continue
				}
				return *secret.ARN, nil
			}
		}
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return "", fmt.Errorf("no sax/backing-services/<UUID> secret found in %s/%s", profile, region)
}

// issueSaxToken runs 'sax issue' and returns the token, or empty string on failure.
func issueSaxToken(sax SaxConfig) string {
	cmd := exec.Command("sax", "issue", "--profile", sax.Profile, "--secret-id", sax.SecretID, "--region", sax.Region)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	// Parse the "Access Token:" header followed by the token on the next line.
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "Access Token:" && i+1 < len(lines) {
			return strings.TrimSpace(lines[i+1])
		}
	}
	return ""
}

// resolveJWTFromCache returns a cached JWT token if available and not expired.
// Uses flock so only one process issues a new token; others wait and read it.
// The token's exp claim (minus a margin) determines freshness.
func resolveJWTFromCache(cell string, sax SaxConfig) string {
	cache := jwtCacheForCell(cell)
	unlock, err := cache.acquireLock()
	if err != nil {
		return ""
	}
	defer unlock()

	if token := cache.read(); token != "" {
		exp := jwtExpTime(token)
		if !exp.IsZero() && time.Until(exp) > jwtExpiryMargin {
			logVerbosef("  Using cached JWT token from %s (expires %s)\n", cache.path, exp.Format(time.RFC3339))
			return token
		}
		logVerbosef("  Cached JWT token in %s expired, issuing a new one...\n", cache.path)
	}

	token := issueSaxToken(sax)
	if token != "" {
		if err := cache.write(token); err != nil {
			fmt.Printf("Warning: failed to cache JWT token to %s: %v\n", cache.path, err)
		} else {
			logVerbosef("  Cached JWT token to %s\n", cache.path)
		}
	}
	return token
}

// ResolveJWTToken returns a JWT token for the given SAX cell.
// If the JWT env variable is set, it is used directly. Otherwise the token
// is obtained via 'sax issue' using credentials discovered from AWS Secrets
// Manager in the cell's region. Both the secret ARN and the JWT token are
// cached per-cell with flock-based locking.
func ResolveJWTToken(cell string) (string, error) {
	jwtToken := os.Getenv("JWT")
	if jwtToken == "" {
		logVerbosef("JWT env variable not set, obtaining fresh token via 'sax issue' for cell %q...\n", cell)
		sax := SaxConfigForCell(cell)
		if sax.SecretID == "" {
			logVerbosef("  SecretID not set, discovering via AWS Secrets Manager (region=%s)...\n", sax.Region)
			arn, err := resolveSecretARN(cell, sax.Profile, sax.Region)
			if err != nil {
				return "", fmt.Errorf("failed to discover SAX secret ARN: %w", err)
			}
			sax.SecretID = arn
			logVerbosef("  Discovered secret ARN: %s\n", arn)
		}
		jwtToken = resolveJWTFromCache(cell, sax)
		if jwtToken == "" {
			// Token failed — cached ARN may be stale; invalidate and re-discover.
			logVerbose("  Cached ARN may be stale, invalidating and re-discovering...")
			arnCacheForCell(cell).invalidate()
			jwtCacheForCell(cell).invalidate()
			arn, err := resolveSecretARN(cell, sax.Profile, sax.Region)
			if err != nil {
				return "", fmt.Errorf("failed to re-discover SAX secret ARN: %w", err)
			}
			if arn != sax.SecretID {
				sax.SecretID = arn
				logVerbosef("  Re-discovered secret ARN: %s\n", arn)
				jwtToken = resolveJWTFromCache(cell, sax)
			}
		}
		if jwtToken == "" {
			fmt.Println("Warning: 'sax issue' did not return a token, check your AWS profile and sax CLI")
		}
	}

	if jwtToken == "" {
		return "", fmt.Errorf("JWT token not found: set JWT env variable or ensure 'sax issue' command works")
	}
	if len(jwtToken) >= 20 {
		logVerbosef("  JWT token loaded   = %s...%s (%d chars)\n", jwtToken[:10], jwtToken[len(jwtToken)-10:], len(jwtToken))
	} else {
		logVerbosef("  JWT token loaded   = %s (%d chars)\n", jwtToken, len(jwtToken))
	}

	return jwtToken, nil
}

// ResolveSAXConfigPath fetches the SAX client config from AWS Secrets Manager
// for the given cell and writes it to a cached temp file. Returns the file path.
// The config is the JSON payload of the backing-service secret (containing
// client_id, private_key, scopes, token_endpoint).
func ResolveSAXConfigPath(cell string) (string, error) {
	sax := SaxConfigForCell(cell)
	cache := saxConfigCacheForCell(cell)

	unlock, err := cache.acquireLock()
	if err != nil {
		return "", fmt.Errorf("acquire sax config cache lock: %w", err)
	}
	defer unlock()

	// Check cache — the config doesn't expire, so any cached value is valid.
	if path := cache.read(); path != "" {
		if _, err := os.Stat(path); err == nil {
			logVerbosef("  Using cached SAX config from %s → %s\n", cache.path, path)
			return path, nil
		}
	}

	// Discover secret ARN (uses its own per-cell cache).
	arn, err := resolveSecretARN(cell, sax.Profile, sax.Region)
	if err != nil {
		return "", fmt.Errorf("failed to discover SAX secret ARN for cell %s: %w", cell, err)
	}

	// Fetch the secret value.
	logVerbosef("  Fetching SAX config secret from %s (region=%s)...\n", arn, sax.Region)
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(sax.Profile),
		config.WithRegion(sax.Region),
	)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := secretsmanager.NewFromConfig(cfg)
	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &arn,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get secret value for %s: %w", arn, err)
	}
	if result.SecretString == nil {
		return "", fmt.Errorf("secret %s has no string value", arn)
	}

	// Write to a temp file.
	tmpFile := filepath.Join(saxCacheDir, fmt.Sprintf("sax-config-%s.json", cell))
	if err := os.WriteFile(tmpFile, []byte(*result.SecretString), 0600); err != nil {
		return "", fmt.Errorf("failed to write SAX config to %s: %w", tmpFile, err)
	}

	// Cache the path.
	if err := cache.write(tmpFile); err != nil {
		fmt.Printf("Warning: failed to cache SAX config path to %s: %v\n", cache.path, err)
	}
	logVerbosef("  SAX config written to %s\n", tmpFile)
	return tmpFile, nil
}

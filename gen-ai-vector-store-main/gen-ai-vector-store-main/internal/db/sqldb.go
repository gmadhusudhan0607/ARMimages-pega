/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	awssecret "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/aws"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	gcpsecret "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/gcp"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	awssecretsmanager "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

const (
	maxElapsedTime = 30 * time.Second
)

type SQLDB interface {
	Begin() (*sql.Tx, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	Close() error
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	Exec(query string, arguments ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, arguments ...any) (sql.Result, error)
	Conn(ctx context.Context) (*sql.Conn, error)
}

type sqlDB struct {
	ctx      context.Context
	dbConfig *config.DatabaseConfig
	dbPool   SQLDB
}

func NewSQLDB(ctx context.Context, dbConfig *config.DatabaseConfig) (SQLDB, error) {
	d := &sqlDB{
		ctx:      ctx,
		dbConfig: dbConfig,
	}

	err := d.loadCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	dbPool, err := d.newDbPool()
	if err != nil {
		return nil, fmt.Errorf("error while connecting to DB with connection string [%s]: %s", dbConfig.ToConnStringMasked(), err.Error())
	}

	d.dbPool = dbPool
	return d, nil

}

// Begin implements pgx.Begin function wrapped by logic to refresh db credentials
func (p *sqlDB) Begin() (*sql.Tx, error) {
	return backoff.RetryWithData[*sql.Tx](func() (*sql.Tx, error) {
		tx, err := p.beginWithPermanentError()
		return tx, p.handleDatabaseError(err)
	}, backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(maxElapsedTime)))
}

func (p *sqlDB) beginWithPermanentError() (*sql.Tx, error) {
	tx, err := p.dbPool.Begin()
	return tx, wrapPermanentError(err)
}

// BeginTx implements context-aware transaction start with backoff retry
func (p *sqlDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	b := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(maxElapsedTime))

	// Wrap with context to respect context cancellation
	backoffWithCtx := backoff.WithContext(b, ctx)

	return backoff.RetryWithData[*sql.Tx](func() (*sql.Tx, error) {
		// Check context before attempting to begin transaction
		if ctx.Err() != nil {
			return nil, backoff.Permanent(ctx.Err())
		}
		tx, err := p.beginTxWithPermanentError(ctx, opts)
		return tx, p.handleDatabaseError(err)
	}, backoffWithCtx)
}

func (p *sqlDB) beginTxWithPermanentError(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	tx, err := p.dbPool.BeginTx(ctx, opts)
	return tx, wrapPermanentError(err)
}

func (p *sqlDB) Close() error {
	return p.dbPool.Close()
}

// Query implements pgx.Query function wrapped by logic to refresh db credentials
func (p *sqlDB) Query(query string, args ...any) (*sql.Rows, error) {
	return backoff.RetryWithData[*sql.Rows](func() (*sql.Rows, error) {
		rows, err := p.queryWithPermanentError(query, args...)
		return rows, p.handleDatabaseError(err)
	}, backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(maxElapsedTime)))
}

func (p *sqlDB) queryWithPermanentError(query string, args ...any) (*sql.Rows, error) {
	rows, err := p.dbPool.Query(query, args...)
	return rows, wrapPermanentError(err)
}

// Exec implements pgx.Exec function wrapped by logic to refresh db credentials
func (p *sqlDB) Exec(query string, args ...any) (sql.Result, error) {
	return backoff.RetryWithData[sql.Result](func() (sql.Result, error) {
		tag, err := p.execWithPermanentError(query, args...)
		return tag, p.handleDatabaseError(err)
	}, backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(maxElapsedTime)))
}

// ExecContext implements pgx.ExecContext function wrapped by logic to refresh db credentials
func (p *sqlDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	b := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(maxElapsedTime))

	// Wrap with context to respect context cancellation
	backoffWithCtx := backoff.WithContext(b, ctx)

	return backoff.RetryWithData[sql.Result](func() (sql.Result, error) {
		// Check context before attempting execution
		if ctx.Err() != nil {
			return nil, backoff.Permanent(ctx.Err())
		}
		tag, err := p.execContextWithPermanentError(ctx, query, args...)
		return tag, p.handleDatabaseError(err)
	}, backoffWithCtx)
}

func (p *sqlDB) execContextWithPermanentError(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tag, err := p.dbPool.ExecContext(ctx, query, args...)
	return tag, wrapPermanentError(err)
}

func (p *sqlDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	b := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(maxElapsedTime))

	// Wrap with context to respect context cancellation
	backoffWithCtx := backoff.WithContext(b, ctx)

	return backoff.RetryWithData[*sql.Rows](func() (*sql.Rows, error) {
		// Check context before attempting query
		if ctx.Err() != nil {
			return nil, backoff.Permanent(ctx.Err())
		}
		rows, err := p.queryContextWithPermanentError(ctx, query, args...)
		return rows, p.handleDatabaseError(err)
	}, backoffWithCtx)
}

func (p *sqlDB) queryContextWithPermanentError(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := p.dbPool.QueryContext(ctx, query, args...)
	return rows, wrapPermanentError(err)
}

func (p *sqlDB) Conn(ctx context.Context) (*sql.Conn, error) {
	b := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(maxElapsedTime))

	// Wrap with context to respect context cancellation
	backoffWithCtx := backoff.WithContext(b, ctx)

	return backoff.RetryWithData(func() (*sql.Conn, error) {
		// Check context before attempting to get connection
		if ctx.Err() != nil {
			return nil, backoff.Permanent(ctx.Err())
		}
		conn, err := p.connWithPermanentError(ctx)
		return conn, p.handleDatabaseError(err)
	}, backoffWithCtx)
}

func (p *sqlDB) connWithPermanentError(ctx context.Context) (*sql.Conn, error) {
	conn, err := p.dbPool.Conn(ctx)
	if err != nil {
		return nil, wrapPermanentError(err)
	}
	return conn, nil
}

func (p *sqlDB) handleDatabaseError(err error) error {
	var perErr backoff.PermanentError
	if err != nil && !perErr.Is(err) {
		// Check if error is due to context cancellation - don't retry
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return backoff.Permanent(err)
		}

		logger.Warn("cannot connect to database, refreshing credentials", zap.Error(err))
		if err := p.refreshCredentials(); err != nil {
			return err
		}
	}
	return err
}

func (p *sqlDB) execWithPermanentError(query string, args ...any) (sql.Result, error) {
	tag, err := p.dbPool.Exec(query, args...)
	return tag, wrapPermanentError(err)
}

func (p *sqlDB) refreshCredentials() (err error) {
	err = p.loadCredentials()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	p.dbPool, err = p.newDbPool()
	if err != nil {
		logger.Error("error while connecting to DB", zap.String("connString", p.dbConfig.ToConnStringMasked()), zap.Error(err))
		return err
	}
	logger.Info("refreshed DB connection string", zap.String("connString", p.dbConfig.ToConnStringMasked()))
	return nil
}

func (p *sqlDB) loadCredentials() (err error) {
	var user, password string

	switch p.dbConfig.Type {
	case config.DatabaseTypeLocal:
		user = helpers.GetEnvOrPanic("DB_USR")
		password = helpers.GetEnvOrPanic("DB_PWD")
	case config.DatabaseTypeCloudAWS:
		sm := awssecretsmanager.NewFromConfig(p.dbConfig.AwsConfig)
		user, password, err = awssecret.GetCredentials(p.ctx, sm)
		if err != nil {
			return fmt.Errorf("failed to get AWS credentials: %w", err)
		}
	case config.DatabaseTypeCloudGCP:
		user, password, err = gcpsecret.GetCredentials(p.ctx)
		if err != nil {
			return fmt.Errorf("failed to get GCP credentials: %w", err)
		}
	default:
		return fmt.Errorf("unknown database type: %d", p.dbConfig.Type)
	}
	p.dbConfig.User = user
	p.dbConfig.Password = password
	return nil
}

// cachingLookupFunc - custom caching DNS resolver (to avoid DNS resolution throttling on a high load)
func cachingLookupFunc(ttl time.Duration, lookupHost func(ctx context.Context, host string) ([]string, error)) func(ctx context.Context, host string) ([]string, error) {
	var (
		cache = make(map[string]struct {
			addrs    []string
			expireAt time.Time
		})
		mu sync.RWMutex
	)
	return func(ctx context.Context, host string) ([]string, error) {
		mu.RLock()
		entry, found := cache[host]
		if found && time.Now().Before(entry.expireAt) {
			mu.RUnlock()
			return entry.addrs, nil
		}
		mu.RUnlock()
		addrs, err := lookupHost(ctx, host)
		if err != nil {
			return nil, err
		}
		mu.Lock()
		cache[host] = struct {
			addrs    []string
			expireAt time.Time
		}{addrs, time.Now().Add(ttl)}
		mu.Unlock()
		return addrs, nil
	}
}

func (p *sqlDB) newDbPool() (SQLDB, error) {
	conf, err := pgx.ParseConfig(p.dbConfig.ToConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to creete pgx.ParseConfig: %w", err)
	}

	// Use custom caching DNS resolver (to avoid DNS resolution throttling on a high load)
	conf.LookupFunc = cachingLookupFunc(5*time.Second, net.DefaultResolver.LookupHost)

	if p.dbConfig.Type == config.DatabaseTypeCloudGCP {
		var opts []cloudsqlconn.Option
		if p.dbConfig.CloudDbWithPrivateIP {
			opts = append(opts, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()))
		}
		d, err := cloudsqlconn.NewDialer(context.Background(), opts...)
		if err != nil {
			return nil, err
		}
		conf.DialFunc = func(ctx context.Context, network, instance string) (net.Conn, error) {
			instanceConnectionName := fmt.Sprintf("%s:%s:%s",
				p.dbConfig.CloudAccount, p.dbConfig.CloudRegion, p.dbConfig.CloudDBInstance)
			return d.Dial(ctx, instanceConnectionName)
		}
	}

	dbURI := stdlib.RegisterConnConfig(conf)
	dbPool, err := sql.Open("pgx", dbURI)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQL connection: %w", err)
	}

	maxCon := int(helpers.GetEnvOrDefaultInt64(p.dbConfig.MaxConnectionEnvVar, p.dbConfig.MaxConnectionsDefault))
	dbPool.SetMaxOpenConns(maxCon)

	return dbPool, nil
}

// wrapPermanentError wraps error to permanent type (ignored by backoff mechanism)
// for other errors then related to authorisation
func wrapPermanentError(err error) error {
	if err != nil {
		var pgErr *pgconn.PgError
		ok := errors.As(err, &pgErr)
		if ok && (pgErr.Code == pgerrcode.InvalidAuthorizationSpecification || pgErr.Code == pgerrcode.InvalidPassword) {
			return err
		}
	}
	return backoff.Permanent(err)
}

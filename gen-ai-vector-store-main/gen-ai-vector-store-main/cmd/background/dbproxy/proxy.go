/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

// For Dev purposes only
// This is a simple proxy server that forwards all incoming connections to a MySQL server.

package dbproxy

import (
	"fmt"
	"io"
	"net"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"go.uber.org/zap"
)

var logger = log.GetNamedLogger("dbproxy")

func NewConnection(host string, port string, conn net.Conn, id uint64) *Connection {
	return &Connection{
		host: host,
		port: port,
		conn: conn,
		id:   id,
	}
}

type Connection struct {
	id   uint64
	conn net.Conn
	host string
	port string
}

func (r *Connection) Handle() error {
	address := fmt.Sprintf("%s%s", r.host, r.port)
	mysql, err := net.Dial("tcp", address)
	if err != nil {
		logger.Error("Failed to connect to MySQL",
			zap.Uint64("connection_id", r.id),
			zap.String("error", err.Error()),
		)
		return err
	}

	go func() {
		copied, err := io.Copy(mysql, r.conn)
		if err != nil {
			logger.Error("Connection error",
				zap.Uint64("connection_id", r.id),
				zap.String("error", err.Error()),
			)
		}
		logger.Info("Connection closed",
			zap.Uint64("connection_id", r.id),
			zap.Int64("bytes_copied", copied),
		)
	}()

	copied, err := io.Copy(r.conn, mysql)
	if err != nil {
		logger.Error("Connection error",
			zap.Uint64("connection_id", r.id),
			zap.String("error", err.Error()),
		)
		return err
	}

	logger.Info("Connection closed",
		zap.Uint64("connection_id", r.id),
		zap.Int64("bytes_copied", copied),
	)
	return nil
}

func NewProxy(host, port string) *Proxy {
	return &Proxy{
		host: host,
		port: port,
	}
}

type Proxy struct {
	host         string
	port         string
	connectionId uint64
}

func (r *Proxy) Start(port string) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		r.connectionId += 1
		logger.Info("Connection accepted",
			zap.Uint64("connection_id", r.connectionId),
			zap.String("remote_addr", conn.RemoteAddr().String()),
		)
		if err != nil {
			logger.Error("Failed to accept new connection",
				zap.Uint64("connection_id", r.connectionId),
				zap.String("error", err.Error()),
			)
			continue
		}

		go r.handle(conn, r.connectionId)
	}
}

func (r *Proxy) handle(conn net.Conn, connectionId uint64) {
	connection := NewConnection(r.host, r.port, conn, connectionId)
	err := connection.Handle()
	if err != nil {
		logger.Error("Error handling proxy connection",
			zap.String("error", err.Error()),
		)
	}
}

/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package schema

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/collections"
)

const serviceName = "genai-vector-store"

// const defaultCacheTTL = 1 * time.Second

type Isolation struct {
	ID          string
	Collections []*Collection
	// cacheTimestamp time.Time
}

type Collection struct {
	IsolationID         string
	CollectionID        string
	SchemaName          string
	TablesPrefix        string
	DefaultEmbProfileID string
	EmbProfiles         map[string]CollectionEmbProfile
	// cacheTimestamp      time.Time
}

type CollectionEmbProfile struct {
	ID     string
	Status string
}

type VsSchemaManager interface {
	Load(ctx context.Context, isolationID, collectionID any) (VsSchemaManager, error)
	GetIsolations() []*Isolation
	GetCollections() []*Collection

	GetIsolation(isolationID string) *Isolation
	IsolationExists(isolationID string) bool
	CollectionExists(isolationID, collectionID string) bool
}

type vsSchemaManager struct {
	database db.Database
	// cacheLock  sync.Mutex
	isolations map[string]*Isolation
	logger     *zap.Logger
}

// var schemaManagerSingleton VsSchemaManager
// var CacheTTL time.Duration = defaultCacheTTL

// func init() {
//  ttlStr := os.Getenv("SCHEMA_CACHE_TTL")
//  if ttlStr != "" {
//      ttlInt, err := strconv.Atoi(ttlStr)
//      if err == nil {
//          logger.Debug("SCHEMA_CACHE_TTL set", zap.Int("seconds", ttlInt))
//          CacheTTL = time.Duration(ttlInt) * time.Second
//      } else {
//          logger.Debug("SCHEMA_CACHE_TTL invalid value", zap.Int("default_seconds", int(defaultCacheTTL.Seconds())), zap.String("invalid_value", ttlStr))
//          CacheTTL = defaultCacheTTL
//      }
//  } else {
//      logger.Debug("SCHEMA_CACHE_TTL default value", zap.Int("seconds", int(defaultCacheTTL.Seconds())))
//      CacheTTL = defaultCacheTTL
//  }
// }

func NewVsSchemaManager(database db.Database, logger *zap.Logger) VsSchemaManager {
	// if schemaManagerSingleton == nil {
	// 	mgr := &vsSchemaManager{
	// 		database:   database,
	// 		isolations: make(map[string]*Isolation),
	// 	}
	// 	schemaManagerSingleton = &tracedVsSchemaManager{next: mgr}
	// }
	// return schemaManagerSingleton

	mgr := &vsSchemaManager{
		database: database,
		logger:   logger,
	}
	return &tracedVsSchemaManager{next: mgr}
}

func (i *Isolation) getCollection(colID string) *Collection {
	for _, c := range i.Collections {
		if c.CollectionID == colID {
			return c
		}
	}
	return nil
}

func (sm *vsSchemaManager) GetIsolations() []*Isolation {
	// logger.Debugf("GetIsolations called")
	// sm.logData()

	isolations := make([]*Isolation, 0, len(sm.isolations))
	for _, iso := range sm.isolations {
		isolations = append(isolations, iso)
	}
	return isolations
}

func (sm *vsSchemaManager) GetCollections() []*Collection {
	// logger.Debugf("GetCollections called")
	// sm.logData()

	collectionList := make([]*Collection, 0)
	for _, iso := range sm.isolations {
		for _, c := range iso.Collections {
			collectionList = append(collectionList, &Collection{
				IsolationID:         c.IsolationID,
				CollectionID:        c.CollectionID,
				SchemaName:          c.SchemaName,
				TablesPrefix:        c.TablesPrefix,
				DefaultEmbProfileID: c.DefaultEmbProfileID,
				EmbProfiles:         c.EmbProfiles,
			})
		}
	}
	return collectionList
}

func (sm *vsSchemaManager) GetIsolation(isolationID string) *Isolation {
	// logger.Debugf("GetIsolation called, isolationID=%s", isolationID)
	// sm.logData()

	return sm.isolations[isolationID]
}

func (sm *vsSchemaManager) getOrCreateIsolation(isolationID string) *Isolation {
	iso, ok := sm.isolations[isolationID]
	if !ok {
		iso = &Isolation{
			ID:          isolationID,
			Collections: make([]*Collection, 0),
			// cacheTimestamp: time.Now(),
		}
		sm.isolations[isolationID] = iso
	}
	return iso
}

func (sm *vsSchemaManager) IsolationExists(isolationID string) bool {
	// logger.Debugf("IsolationExists called, isolationID=%s", isolationID)
	// sm.logData()

	_, ok := sm.isolations[isolationID]
	sm.logger.Debug("IsolationExists result", zap.Bool("exists", ok))
	return ok
}

func (sm *vsSchemaManager) CollectionExists(isolationID, collectionName string) bool {
	// logger.Debugf("CollectionExists called, isolationID=%s, collectionName=%s", isolationID, collectionName)
	// sm.logData()

	iso, ok := sm.isolations[isolationID]
	if !ok {
		sm.logger.Debug("Isolation does not exist", zap.String("isolationID", isolationID))
		return false
	}
	for _, c := range iso.Collections {
		if c.CollectionID == collectionName {
			sm.logger.Debug("Collection exists in isolation", zap.String("collectionID", collectionName), zap.String("isolationID", isolationID))
			return true
		}
	}
	sm.logger.Debug("Collection does not exist in isolation", zap.String("collectionID", collectionName), zap.String("isolationID", isolationID))
	return false
}

func (sm *vsSchemaManager) Load(ctx context.Context, isolationID, collectionID any) (VsSchemaManager, error) {
	// logger.Debugf("Load called, isolationID=%v, collectionID=%v", isolationID, collectionID)
	// sm.logData()

	// sm.cacheLock.Lock()
	// defer sm.cacheLock.Unlock()

	// if sm.isolations == nil {
	// 	sm.isolations = make(map[string]*Isolation)
	// }

	// isoIDStr, _ := isolationID.(string)
	// colIDStr, _ := collectionID.(string)
	// if isoIDStr != "" {
	// 	return sm.loadByIsolationID(ctx, isoIDStr, colIDStr)
	// }
	// return sm.loadAllIsolations(ctx)

	sm.isolations = make(map[string]*Isolation)

	query := fmt.Sprintf(`
		SELECT COALESCE(iso_id, ''), 
		       COALESCE(col_id, ''),
		       COALESCE(profile_id,''), 
		       COALESCE(schema_name,''),
		       COALESCE(tables_prefix,''),
		       COALESCE(profile_status,'%s'),
		       COALESCE(is_default_profile, false)
		FROM vector_store.schema_info($1, $2)`, collections.EmbeddingProfileStatusUnknown)

	rows, err := sm.database.GetConn().Query(query, isolationID, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema info [%s]: %w", query, err)
	}
	defer rows.Close()

	// Process the result rows
	for rows.Next() {
		var isoID, colID, profID, schema, prefix, status string
		var isDefault bool
		err = rows.Scan(&isoID, &colID, &profID, &schema, &prefix, &status, &isDefault)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schema info [%s]: %w", query, err)
		}

		if isoID != "" {
			iso := sm.getOrCreateIsolation(isoID)
			if colID != "" {
				if err = sm.updateCollectionData(iso, colID, schema, prefix, profID, status, isDefault); err != nil {
					return nil, err
				}
			}
		}
	}
	return sm, rows.Err()
}

// func (sm *vsSchemaManager) loadByIsolationID(ctx context.Context, isoIDStr, colIDStr string) (VsSchemaManager, error) {
// 	iso, ok := sm.isolations[isoIDStr]
// 	if !ok || CacheTTL == 0 {
// 		// Isolation doesn't in cache or cache disabled, always load full isolation
// 		logger.Debugf("Isolation doesn't in cache or cache disabled, always load full isolation: %s", isoIDStr)
// 		return sm.loadFullIsolationFromDB(ctx, isoIDStr)
// 	}

// 	if colIDStr == "" {
// 		// No collection specified, always reload full isolation as there might be changes.
// 		// New collections might have been added.
// 		logger.Debugf("No collection specified. New collections might have been added. Reload full isolation: %s", isoIDStr)
// 		return sm.loadFullIsolationFromDB(ctx, isoIDStr)
// 	}

// 	// colIDStr is provided
// 	col := iso.getCollection(colIDStr)
// 	if col == nil {
// 		// Collection isn't found in cache, reload full isolation, As it might have been added since last load
// 		logger.Debugf("Collection %s not found in isolation %s. It might have been added since last load. Reloading full isolation from DB", colIDStr, isoIDStr)
// 		return sm.loadFullIsolationFromDB(ctx, isoIDStr)
// 	}
// 	if time.Since(col.cacheTimestamp) < CacheTTL {
// 		logger.Debugf("Returning cached isolation %s and collection %s", isoIDStr, colIDStr)
// 		sm.logData()
// 		return sm, nil
// 	}
// 	// Collection expired, reload full isolation
// 	logger.Debugf("Collection %s in isolation %s expired, reloading full isolation from DB", colIDStr, isoIDStr)
// 	return sm.loadFullIsolationFromDB(ctx, isoIDStr)
// }

// func (sm *vsSchemaManager) loadFullIsolationFromDB(ctx context.Context, isoIDStr string) (VsSchemaManager, error) {
// 	isoMap, err := sm.loadIsoFromDB(ctx, isoIDStr)
// 	if err != nil {
// 		logger.Errorf("Failed to load isolation %s from DB: %v", isoIDStr, err)
// 		sm.logData()
// 		return nil, err
// 	}
// 	for id, newIso := range isoMap {
// 		newIso.cacheTimestamp = time.Now()
// 		sm.isolations[id] = newIso
// 		logger.Debugf("Updated isolation %s in cache", id)
// 	}
// 	logger.Debugf("Loaded isolation %s from DB", isoIDStr)
// 	sm.logData()
// 	return sm, nil
// }

// func (sm *vsSchemaManager) loadAllIsolations(ctx context.Context) (VsSchemaManager, error) {
// 	isoMap, err := sm.loadIsoFromDB(ctx, nil)
// 	if err != nil {
// 		logger.Errorf("Failed to load all isolations from DB: %v", err)
// 		return nil, err
// 	}
// 	for _, iso := range isoMap {
// 		iso.cacheTimestamp = time.Now()
// 		sm.isolations[iso.ID] = iso
// 		logger.Debugf("Updated isolation %s in cache", iso.ID)
// 	}
// 	logger.Debugf("Loaded all isolations from DB")
// 	sm.logData()
// 	return sm, nil
// }

// // Helper to load a single isolation from DB
// func (sm *vsSchemaManager) loadIsoFromDB(ctx context.Context, isolationID any) (map[string]*Isolation, error) {
// 	query := fmt.Sprintf(`
// 		SELECT COALESCE(iso_id, ''),
// 		       COALESCE(col_id, ''),
// 		       COALESCE(profile_id,''),
// 		       COALESCE(schema_name,''),
// 		       COALESCE(tables_prefix,''),
// 		       COALESCE(profile_status,'%s'),
// 		       COALESCE(is_default_profile, false)
// 		FROM vector_store.schema_info($1, $2)`, collections.EmbeddingProfileStatusUnknown)

// 	rows, err := sm.database.GetConn().Query(query, isolationID, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read schema info [%s]: %w", query, err)
// 	}
// 	defer rows.Close()

// 	isolations := make(map[string]*Isolation)
// 	for rows.Next() {
// 		var isoID, colID, profID, schema, prefix, status string
// 		var isDefault bool
// 		err = rows.Scan(&isoID, &colID, &profID, &schema, &prefix, &status, &isDefault)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to scan schema info [%s]: %w", query, err)
// 		}

// 		if isoID != "" {
// 			iso := sm.getOrCreateIsolation(isoID)
// 			isolations[isoID] = iso
// 			if colID != "" {
// 				if err = sm.updateCollectionData(iso, colID, schema, prefix, profID, status, isDefault); err != nil {
// 					return nil, err
// 				}
// 			}
// 		}
// 	}
// 	return isolations, nil
// }

func (sm *vsSchemaManager) updateCollectionData(iso *Isolation, colID, schema, prefix, profID, status string, isDefault bool) error {

	if colID == "" || schema == "" || prefix == "" {
		return fmt.Errorf("failed to update collection: empty value found: colID=%s, schema=%s, prefix=%s",
			colID, schema, prefix)
	}

	col := iso.getCollection(colID)
	if col == nil {
		col = &Collection{
			IsolationID:  iso.ID,
			CollectionID: colID,
			SchemaName:   schema,
			TablesPrefix: prefix,
			EmbProfiles:  make(map[string]CollectionEmbProfile),
			// cacheTimestamp: time.Now(),
		}
		iso.Collections = append(iso.Collections, col)
	} else {
		col.SchemaName = schema
		col.TablesPrefix = prefix
		// col.cacheTimestamp = time.Now()
	}

	if profID != "" {
		col.EmbProfiles[profID] = CollectionEmbProfile{
			ID:     profID,
			Status: status,
		}
	}

	// Set default profile if applicable
	if isDefault {
		col.DefaultEmbProfileID = profID
	}
	return nil
}

// func (sm *vsSchemaManager) logData() {
// 	logger.Debugf("==++ CACHE CONTENT ++==")
// 	for isoID, iso := range sm.isolations {
// 		logger.Debugf("    ==> Isolation: %s, time before expiration: %s\n", isoID, time.Until(iso.cacheTimestamp.Add(CacheTTL)))
// 		for _, col := range iso.Collections {
// 			logger.Debugf("      ==> Collection: %s, Schema: %s, Prefix: %s\n", col.CollectionID, col.SchemaName, col.TablesPrefix)
// 		}
// 	}
// 	logger.Debugf("=++ +++++++++++++ ++==")

// }

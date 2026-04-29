/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package opsmetrics

import (
	"fmt"
	"slices"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
)

type CollectionMetrics struct {
	documentsCount        int64
	diskUsage             int64
	documentsModification *time.Time
}

func (o *opsMetrics) GetCollectionsMetrics(metricName []string) (result []CollectionsMetrics, err error) {

	colMetricsList, err := o.getCollectionMetrics(metricName)
	if err != nil {
		return nil, fmt.Errorf("error while getting collection metrics: %w", err)
	}
	for colID, colMetrics := range colMetricsList {
		result = append(result,
			CollectionsMetrics{
				ID: colID,
				DocumentsMetrics: DocumentsMetrics{
					DiskUsage:             colMetrics.diskUsage,
					DocumentsCount:        colMetrics.documentsCount,
					DocumentsModification: colMetrics.documentsModification,
				}})
	}
	return result, nil
}

func (o *opsMetrics) GetIsolationMetrics(metricName []string) (*DocumentsMetrics, error) {
	isoMetrics := &DocumentsMetrics{}
	colMetricsList, err := o.getCollectionMetrics(metricName)
	if err != nil {
		return nil, fmt.Errorf("error while getting collection metrics: %w", err)
	}
	for _, colMetrics := range colMetricsList {
		isoMetrics.DocumentsCount += colMetrics.documentsCount
		isoMetrics.DiskUsage += colMetrics.diskUsage
		if isoMetrics.DocumentsModification == nil {
			isoMetrics.DocumentsModification = colMetrics.documentsModification
		} else {
			if colMetrics.documentsModification.After(*isoMetrics.DocumentsModification) {
				isoMetrics.DocumentsModification = colMetrics.documentsModification
			}
		}
	}
	return isoMetrics, nil

}

func (o *opsMetrics) getCollectionMetrics(metricNames []string) (cmList map[string]CollectionMetrics, err error) {

	colMetricsList := map[string]CollectionMetrics{}

	var documentsCountList map[string]int64
	var diskUsageList map[string]int64
	var modificationTimeList map[string]*time.Time

	if len(metricNames) == 0 || slices.Contains(metricNames, "documentsCount") {
		documentsCountList, err = o.getDocumentsCount()
		if err != nil {
			return nil, fmt.Errorf("error while getting document count: %w", err)
		}
	}

	if len(metricNames) == 0 || slices.Contains(metricNames, "diskUsage") {
		diskUsageList, err = o.getDiskUsage()
		if err != nil {
			return nil, fmt.Errorf("error while getting discUsage count: %w", err)
		}
	}

	if len(metricNames) == 0 || slices.Contains(metricNames, "documentsModification") {
		modificationTimeList, err = o.getModificationTime()
		if err != nil {
			return nil, fmt.Errorf("error while getting documentsModification count: %w", err)
		}
	}

	colIDs, err := o.getCollectionNames()
	if err != nil {
		return nil, fmt.Errorf("error while getting collection names: %w", err)
	}

	for _, colID := range colIDs {
		colMetricsList[colID] = CollectionMetrics{
			documentsCount:        documentsCountList[colID],
			diskUsage:             diskUsageList[colID],
			documentsModification: modificationTimeList[colID],
		}
	}
	return colMetricsList, nil
}

func (o *opsMetrics) getDocumentsCount() (docCountPerCollection map[string]int64, err error) {
	docCountPerCollection = map[string]int64{}
	query := "SELECT * FROM vector_store.metrics_document_count($1)"
	rows, err := o.database.GetConn().Query(query, o.isolationID)
	if err != nil {
		return nil, fmt.Errorf("error while executing query [%s] : %w", query, err)
	}
	defer rows.Close()
	for rows.Next() {
		var collectionID string
		var count int64
		err = rows.Scan(&collectionID, &count)
		if err != nil {
			return nil, fmt.Errorf("error while scanning rows: %w", err)
		}
		docCountPerCollection[collectionID] = count
	}
	return docCountPerCollection, nil
}

func (o *opsMetrics) getDiskUsage() (discUsagePerCollection map[string]int64, err error) {
	discUsagePerCollection = map[string]int64{}
	query := `SELECT col_id, coalesce(sum(table_size),0) FROM vector_store.metrics_table_size($1) WHERE col_id <>'' GROUP BY col_id`
	rows, err := o.database.GetConn().Query(query, o.isolationID)
	if err != nil {
		return nil, fmt.Errorf("error while executing query [%s] : %w", query, err)
	}
	defer rows.Close()
	for rows.Next() {
		var colID string
		var size int64
		err = rows.Scan(&colID, &size)
		if err != nil {
			return nil, fmt.Errorf("error while scanning rows: %w", err)
		}
		discUsagePerCollection[colID] = size
	}
	return discUsagePerCollection, nil
}

func (o *opsMetrics) getModificationTime() (modificationTimePerCollection map[string]*time.Time, err error) {
	modificationTimePerCollection = map[string]*time.Time{}
	query := "SELECT * FROM vector_store.metrics_last_modified_time($1)"
	rows, err := o.database.GetConn().Query(query, o.isolationID)
	if err != nil {
		return nil, fmt.Errorf("error while executing query [%s] : %w", query, err)
	}
	defer rows.Close()
	for rows.Next() {
		var colID string
		var mTime time.Time
		err = rows.Scan(&colID, &mTime)
		if err != nil {
			return nil, fmt.Errorf("error while scanning rows: %w", err)
		}
		modificationTimePerCollection[colID] = &mTime
	}
	return modificationTimePerCollection, nil
}

func (o *opsMetrics) getCollectionNames() (colIDs []string, err error) {
	query := fmt.Sprintf("SELECT col_id FROM vector_store_%s.collections", db.GetMD5Hash(o.isolationID))
	rows, err := o.database.GetConn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("error while executing query [%s] : %w", query, err)
	}
	defer rows.Close()
	for rows.Next() {
		var colID string
		err = rows.Scan(&colID)
		if err != nil {
			return nil, fmt.Errorf("error while scanning rows: %w", err)
		}
		colIDs = append(colIDs, colID)
	}
	return colIDs, nil
}

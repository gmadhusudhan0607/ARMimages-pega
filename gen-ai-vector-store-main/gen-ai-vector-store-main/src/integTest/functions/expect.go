/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package test_functions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/google/go-cmp/cmp"
	"github.com/pgvector/pgvector-go"

	db2 "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func ExpectServiceIsAccessible(uri string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect service is accesible on %s", uri))
	u, err := url.Parse(uri)
	Expect(err).To(BeNil())
	Expect(isPortAccessible(u.Hostname(), u.Port())).To(Equal(true), fmt.Sprintf("Service is not accessible on %s", uri))
}

func ExpectIsolationExistsInDbWithMaxStorageSize(ctx context.Context, db *pgxpool.Pool, isolationID, maxStorageSize string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect isolation %s exists in DB with maxStorageSize=%s ", isolationID, maxStorageSize))
	var id string
	var size string
	sqlQuery := "SELECT iso_id, max_storage_size FROM vector_store.isolations WHERE iso_id=$1"
	err := db.QueryRow(ctx, sqlQuery, isolationID).Scan(&id, &size)
	Expect(err).To(BeNil())
	Expect(id).To(Equal(isolationID))
	Expect(size).To(Equal(maxStorageSize))
}

func ExpectIsolationExistsInDB(ctx context.Context, db *pgxpool.Pool, isolationID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect isolation %s exists in DB", isolationID))
	var id string
	sqlQuery := "SELECT iso_id FROM vector_store.isolations WHERE iso_id=$1"
	err := db.QueryRow(ctx, sqlQuery, isolationID).Scan(&id)
	Expect(err).To(BeNil())
	Expect(id).To(Equal(isolationID))
}

func ExpectIsolationDoesNotExistInDB(ctx context.Context, db *pgxpool.Pool, isolationID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect isolation %s does not exist in DB", isolationID))
	var count int
	sqlQuery := "SELECT COUNT(*) FROM vector_store.isolations WHERE iso_id=$1"
	err := db.QueryRow(ctx, sqlQuery, isolationID).Scan(&count)
	Expect(err).To(BeNil())
	Expect(count).To(Equal(0))
}

func ExpectTableExistsInDB(ctx context.Context, db *pgxpool.Pool, tableName string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect table %s exists in DB", tableName))
	var result bool
	schema, table := helpers.SplitTableName(tableName)
	sqlQuery := `SELECT EXISTS ( SELECT FROM pg_tables WHERE schemaname = $1 AND tablename = $2 )`
	err := db.QueryRow(ctx, sqlQuery, schema, table).Scan(&result)
	Expect(err).To(BeNil())
	Expect(result).To(Equal(true))
}

func ExpectCollectionEmbeddingProfileExists(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, profileID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect collection '%s/%s' has profile '%s'", isolationID, collectionID, profileID))
	var count int
	tableName := db2.GetTableCollectionEmbeddingProfiles(isolationID)
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE col_id=$1 AND profile_id=$2", tableName)
	rows := db.QueryRow(ctx, query, collectionID, profileID)
	Expect(rows).NotTo(BeNil())
	err := rows.Scan(&count)
	Expect(err).To(BeNil())
	Expect(count).To(Equal(1))
}

func ExpectCollectionEmbeddingProfileDoesNotExist(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, profileID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect collection '%s/%s' has profile '%s'", isolationID, collectionID, profileID))
	var count int
	tableName := db2.GetTableCollectionEmbeddingProfiles(isolationID)
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE col_id=$1 AND profile_id=$2", tableName)
	rows := db.QueryRow(ctx, query, collectionID, profileID)
	Expect(rows).NotTo(BeNil())
	err := rows.Scan(&count)
	Expect(err).To(BeNil())
	Expect(count).To(Equal(0))
}

func ExpectTableDoesNotExistInDB(ctx context.Context, db *pgxpool.Pool, tableName string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect table %s does not exist in DB", tableName))
	var result bool
	schema, table := helpers.SplitTableName(tableName)
	sqlQuery := `SELECT EXISTS ( SELECT FROM pg_tables WHERE schemaname = $1 AND tablename = $2 )`
	err := db.QueryRow(ctx, sqlQuery, schema, table).Scan(&result)
	Expect(err).To(BeNil())
	Expect(result).To(Equal(false))
}

func ExpectResponseMatchFromFile(response, responseFile string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect response match the one from file %s", responseFile))

	got := TrimAllSpacesDuplicates(response)
	expected := TrimAllSpacesDuplicates(ReadTestDataFile(responseFile))
	Expect(got).To(MatchJSON(expected))
}

func ExpectResponseMatchFromFileIgnoringFields(response, responseFile string, ignoredFields ...string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect response match the one from file %s, ignoring fields: %v", responseFile, ignoredFields))

	var actual, expected interface{}
	// Unmarshal the actual response (can be object or array)
	err := json.Unmarshal([]byte(response), &actual)
	Expect(err).To(BeNil(), "Failed to unmarshal actual response")

	// Unmarshal the expected response from file
	expectedContent := ReadTestDataFile(responseFile)
	err = json.Unmarshal([]byte(expectedContent), &expected)
	Expect(err).To(BeNil(), "Failed to unmarshal expected response")

	// Recursively remove ignored fields from both actual and expected responses
	removeIgnoredFields(actual, ignoredFields...)
	removeIgnoredFields(expected, ignoredFields...)

	// Compare the modified responses
	Expect(actual).To(Equal(expected), "Responses do not match after ignoring specified fields")
}

func removeIgnoredFields(data interface{}, ignoredFields ...string) {
	ignoredSet := make(map[string]struct{}, len(ignoredFields))
	for _, field := range ignoredFields {
		ignoredSet[field] = struct{}{}
	}

	var removeFields func(v interface{})
	removeFields = func(v interface{}) {
		if v == nil {
			return
		}

		switch obj := v.(type) {
		case map[string]interface{}:
			for key := range obj {
				if _, ignored := ignoredSet[key]; ignored {
					delete(obj, key)
					continue
				}
				removeFields(obj[key])
			}
		case []interface{}:
			for i := range obj {
				removeFields(obj[i])
			}
		}
	}

	removeFields(data)
}

func ExpectFirstItemHasNonEmptyFields(response string, fields ...string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect first item has non-empty fields: %v", fields))

	var items []map[string]interface{}
	err := json.Unmarshal([]byte(response), &items)
	Expect(err).To(BeNil(), "Failed to unmarshal response into slice")
	Expect(items).NotTo(BeEmpty(), "Expected at least one item in response")

	first := items[0]
	for _, field := range fields {
		Expect(first).To(HaveKey(field))
		Expect(first[field]).NotTo(BeEmpty())
	}
}

// ExpectResponseFieldsMatchFromFile compares only the JSON structure (field names and nesting),
// not the concrete primitive values. It passes as long as the actual response has the same
// object/array shape and keys as the expected JSON loaded from file.
func ExpectResponseFieldsMatchFromFile(response, responseFile string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect response fields match the one from file %s", responseFile))

	var actual, expected interface{}

	err := json.Unmarshal([]byte(response), &actual)
	Expect(err).To(BeNil(), "Failed to unmarshal actual response")

	expectedContent := ReadTestDataFile(responseFile)
	err = json.Unmarshal([]byte(expectedContent), &expected)
	Expect(err).To(BeNil(), "Failed to unmarshal expected response")

	Expect(jsonStructureEqual(actual, expected)).To(BeTrue(), "Response JSON structure (fields) does not match expected")
}

// jsonStructureEqual returns true if both JSON values have the same structural shape:
// - same object keys
// - same array lengths
// - primitive types in corresponding positions (values themselves are not compared)
func jsonStructureEqual(a, b interface{}) bool {
	switch av := a.(type) {
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for k, avv := range av {
			bvv, ok := bv[k]
			if !ok {
				return false
			}
			if !jsonStructureEqual(avv, bvv) {
				return false
			}
		}
		return true
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonStructureEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		// For primitive values, just require that both are non-container types
		switch b.(type) {
		case map[string]interface{}, []interface{}:
			return false
		default:
			return true
		}
	}
}

// SaveResponseToFile writes the given JSON response string into a testdata file path,
// preserving pretty formatting when possible. This is useful for (re)generating
// golden response files during test development.
func SaveResponseToFile(response, responseFile string) {
	GinkgoHelper()

	By(fmt.Sprintf("Save response into testdata file %s", responseFile))

	var js interface{}
	if err := json.Unmarshal([]byte(response), &js); err != nil {
		// If it's not valid JSON, still write raw content for debugging
		Expect(os.WriteFile(testDataFullPath(responseFile), []byte(response), 0o644)).To(Succeed())
		return
	}

	formatted, err := json.MarshalIndent(js, "", "  ")
	Expect(err).To(BeNil())
	Expect(os.WriteFile(testDataFullPath(responseFile), formatted, 0o644)).To(Succeed())
}

// testDataFullPath resolves a path under the current integTest data/ directory,
// consistent with ReadTestDataFile.
func testDataFullPath(file string) string {
	path, err := os.Getwd()
	Expect(err).To(BeNil())
	return fmt.Sprintf("%s/data/%s", path, file)
}

func ExpectSortedJSONEqualsFromFile(response, responseFile, param string) {
	GinkgoHelper()
	By(fmt.Sprintf("Expect response match the one from file %s and unmarshal", responseFile))
	respFromFile := ReadTestDataFile(responseFile)

	ExpectSortedJSONEquals([]byte(response), []byte(respFromFile), param)
}

func ExpectArrayItemsHaveFieldsFromFile(response, responseFile string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect each array item has required fields defined in %s (values and order are ignored)", responseFile))

	var actualItems []map[string]interface{}
	var expectedItems []map[string]interface{}

	// Unmarshal actual response
	if err := json.Unmarshal([]byte(response), &actualItems); err != nil {
		Expect(err).To(BeNil(), "Failed to unmarshal actual response as array of objects")
	}
	Expect(actualItems).NotTo(BeEmpty(), "Actual response array is empty")

	// Unmarshal expected response from file
	expectedContent := ReadTestDataFile(responseFile)
	if err := json.Unmarshal([]byte(expectedContent), &expectedItems); err != nil {
		Expect(err).To(BeNil(), "Failed to unmarshal expected response as array of objects")
	}
	Expect(expectedItems).NotTo(BeEmpty(), "Expected response array is empty")

	// Build the set of required keys as the intersection of keys
	// present in all expected items. This allows some fields (like "error")
	// to be optional and present only on a subset of items.
	if len(expectedItems) == 0 {
		return
	}

	requiredKeys := make(map[string]struct{})
	// Initialize with keys from the first expected item.
	for key := range expectedItems[0] {
		requiredKeys[key] = struct{}{}
	}

	// Intersect with keys from the remaining expected items.
	for _, exp := range expectedItems[1:] {
		for key := range requiredKeys {
			if _, ok := exp[key]; !ok {
				delete(requiredKeys, key)
			}
		}
	}

	// For every actual item, ensure it has at least the required keys.
	for i, act := range actualItems {
		for key := range requiredKeys {
			if _, ok := act[key]; !ok {
				Expect(ok).To(BeTrue(), fmt.Sprintf("Item %d is missing required field '%s'", i, key))
			}
		}
	}
}

// ExpectFindDocumentsArrayItemsHaveFieldsFromFile is similar to ExpectArrayItemsHaveFieldsFromFile
// but tailored for the find-documents v2 API, whose response is an object envelope
// of the form: {"documents": [...], "pagination": {...}}.
// It extracts the documents array from both actual and expected JSON and then
// asserts that each actual document has at least the keys present in the expected
// documents (values and order are ignored).
func ExpectFindDocumentsArrayItemsHaveFieldsFromFile(response, responseFile string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect each document item has required fields defined in %s (values and order are ignored)", responseFile))

	type envelope struct {
		Documents []map[string]interface{} `json:"documents"`
	}

	var actualEnv envelope
	var expectedEnv envelope

	// Unmarshal actual response (full envelope)
	if err := json.Unmarshal([]byte(response), &actualEnv); err != nil {
		Expect(err).To(BeNil(), "Failed to unmarshal actual response as find-documents envelope")
	}
	Expect(actualEnv.Documents).NotTo(BeEmpty(), "Actual documents array is empty")

	// Unmarshal expected response from file (full envelope)
	expectedContent := ReadTestDataFile(responseFile)
	if err := json.Unmarshal([]byte(expectedContent), &expectedEnv); err != nil {
		Expect(err).To(BeNil(), "Failed to unmarshal expected response as find-documents envelope")
	}
	Expect(expectedEnv.Documents).NotTo(BeEmpty(), "Expected documents array is empty")

	// Build the set of required keys as the intersection of keys
	// present in all expected documents. This allows some fields
	// (like documentAttributes) to be optional and present only
	// on a subset of documents.
	if len(expectedEnv.Documents) == 0 {
		return
	}

	requiredKeys := make(map[string]struct{})
	// Initialize with keys from the first expected document.
	for key := range expectedEnv.Documents[0] {
		requiredKeys[key] = struct{}{}
	}

	// Intersect with keys from the remaining expected documents.
	for _, exp := range expectedEnv.Documents[1:] {
		for key := range requiredKeys {
			if _, ok := exp[key]; !ok {
				delete(requiredKeys, key)
			}
		}
	}

	// For every actual document, ensure it has at least the required keys.
	for i, act := range actualEnv.Documents {
		for key := range requiredKeys {
			if _, ok := act[key]; !ok {
				Expect(ok).To(BeTrue(), fmt.Sprintf("Document %d is missing required field '%s'", i, key))
			}
		}
	}
}

func ExpectJSONEqualsFromFile(response, responseFile string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect response match the one from file %s and unmarshal", responseFile))
	respFromFile := ReadTestDataFile(responseFile)

	ExpectJSONEquals([]byte(response), []byte(respFromFile))
}

func ExpectDocumentStatusInDB(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID, status string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect document %s/%s/%s status='%s'", isolationID, collectionID, docID, status))
	docStatus, _ := GetDocumentStatusAndErrorFromDB(ctx, db, isolationID, collectionID, docID)
	Expect(docStatus).To(Equal(status))
}

func ExpectDocumentErrorInDB(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID, err string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect document %s/%s/%s error='%s'", isolationID, collectionID, docID, err))
	_, docErr := GetDocumentStatusAndErrorFromDB(ctx, db, isolationID, collectionID, docID)
	Expect(docErr).To(Equal(err), fmt.Sprintf("JSONs are not equal:\n%s", cmp.Diff(docErr, err)))
}

// ExpectDocumentErrorInDBOneOf asserts that the document error matches one of the provided expected strings.
// Use this when parallel chunk embedding may produce different but equally valid error states depending
// on goroutine scheduling: e.g. both chunks may record an error (count:2) or one may be cancelled
// before recording (count:1 with the other staying IN_PROGRESS).
func ExpectDocumentErrorInDBOneOf(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID string, possibleErrors ...string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect document %s/%s/%s error matches one of %d possible states", isolationID, collectionID, docID, len(possibleErrors)))
	_, docErr := GetDocumentStatusAndErrorFromDB(ctx, db, isolationID, collectionID, docID)
	for _, expected := range possibleErrors {
		if docErr == expected {
			return
		}
	}
	Fail(fmt.Sprintf("Document error %q did not match any of the expected values: %s", docErr, strings.Join(possibleErrors, ", ")))
}

// ExpectSortedJSONEqualsFromOneOfFiles asserts that the response matches one of the provided fixture files
// after sorting by param. Use this when the response may have multiple valid representations.
func ExpectSortedJSONEqualsFromOneOfFiles(response, param string, responseFiles ...string) {
	GinkgoHelper()

	for _, file := range responseFiles {
		respFromFile := ReadTestDataFile(file)
		j1, j2, err := GetSortedJSON([]byte(response), []byte(respFromFile), param)
		if err == nil && SortedJSONEquals(j1, j2) {
			By(fmt.Sprintf("Response matches file %s", file))
			return
		}
	}
	Fail(fmt.Sprintf("Response did not match any of the expected files: %s", strings.Join(responseFiles, ", ")))
}

// ExpectJSONEqualsFromOneOfFiles asserts that the response matches one of the provided fixture files.
// Use this when the response may have multiple valid representations.
func ExpectJSONEqualsFromOneOfFiles(response string, responseFiles ...string) {
	GinkgoHelper()

	for _, file := range responseFiles {
		respFromFile := ReadTestDataFile(file)
		var d1, d2 map[string]interface{}
		if json.Unmarshal([]byte(response), &d1) == nil && json.Unmarshal([]byte(respFromFile), &d2) == nil {
			if JSONEquals(d1, d2) {
				By(fmt.Sprintf("Response matches file %s", file))
				return
			}
		}
	}
	Fail(fmt.Sprintf("Response did not match any of the expected files: %s", strings.Join(responseFiles, ", ")))
}

func ExpectChunksCountInDatabase(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID string, count int) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect chunks count in database %s/%s/%s is %d", isolationID, collectionID, docID, count))
	rowsCount := GetChunksCountFromDB(ctx, db, isolationID, collectionID, docID)
	Expect(rowsCount).To(Equal(count))
}

func ExpectChunksProcessingCountInDatabase(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID string, count int) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect chunks processing count in database %s/%s/%s is %d", isolationID, collectionID, docID, count))
	rowsCount := GetChunksProcessingCountFromDB(ctx, db, isolationID, collectionID, docID)
	Expect(rowsCount).To(Equal(count))
}

func ExpectEmbeddingsEmptyInDatabase(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect embeddings empty in database %s/%s/%s", isolationID, collectionID, docID))

	tableEmb := db2.GetTableEmb(isolationID, collectionID)
	tableAttr := db2.GetTableAttr(isolationID, collectionID)
	sqlQuery := fmt.Sprintf(`
		SELECT doc_id, emb_id, content, embedding,
			   vector_store.attributes_as_jsonb_by_ids('%[2]s', attr_ids ) as attributes
		FROM %[1]s  WHERE doc_id=$1
	`, tableEmb, tableAttr)
	rows, err := db.Query(ctx, sqlQuery, docID)
	Expect(err).To(BeNil())
	defer rows.Close()

	for i := 0; rows.Next(); i++ {
		chunk := embedings.Chunk{}
		err = rows.Scan(&chunk.Content, &chunk.Vector)
		Expect(err).ToNot(BeNil())
		Expect(chunk.Vector).To(Equal(pgvector.Vector{}))
	}
}

func ExpectEmbeddingsProcessingEmptyInDatabase(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect embeddings processing empty in database %s/%s/%s", isolationID, collectionID, docID))

	tableEmbProcessing := db2.GetTableEmbProcessing(isolationID, collectionID)
	sqlQuery := fmt.Sprintf(`
		SELECT embedding
		FROM %[1]s  WHERE doc_id=$1
	`, tableEmbProcessing)
	rows, err := db.Query(ctx, sqlQuery, docID)
	Expect(err).To(BeNil())
	defer rows.Close()

	for i := 0; rows.Next(); i++ {
		chunk := embedings.Chunk{}
		err = rows.Scan(&chunk.Vector)
		Expect(err).ToNot(BeNil())
		Expect(chunk.Vector).To(Equal(pgvector.Vector{}))
	}
}

func ExpectEmbeddingsInDatabase(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID string, expectedEmbedding []embedings.Embedding) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect chunks content in database %s/%s/%s", isolationID, collectionID, docID))
	tableEmb := db2.GetTableEmb(isolationID, collectionID)
	tableAttr := db2.GetTableAttr(isolationID, collectionID)
	sqlQuery := fmt.Sprintf(`
		SELECT doc_id, emb_id, content, embedding, 
			   vector_store.attributes_as_jsonb_by_ids('%[2]s', attr_ids ) as attributes,
			   vector_store.attributes_as_jsonb_by_ids('%[2]s', attr_ids2 ) as attributes
		FROM %[1]s  WHERE doc_id=$1
	`, tableEmb, tableAttr)
	rows, err := db.Query(ctx, sqlQuery, docID)
	Expect(err).To(BeNil())
	defer rows.Close()

	retrievedEmbedding := make([]embedings.Embedding, 0)
	for i := 0; rows.Next(); i++ {
		emb := embedings.Embedding{}
		err = rows.Scan(&emb.DocumentID, &emb.ID, &emb.Content, &emb.Vector, &emb.Attributes, &emb.Attributes2)
		Expect(err).To(BeNil())
		retrievedEmbedding = append(retrievedEmbedding, emb)
	}

	for _, ec := range expectedEmbedding {
		found := false
		for _, rc := range retrievedEmbedding {
			if rc.ID == ec.ID {
				found = true
				ExpectEmbeddingAreEqual(ec, rc)
			}
		}
		Expect(found).To(BeTrue(), fmt.Sprintf("Embedding '%s' not found in database", ec.ID))
	}

}

func ExpectEmbeddingAreEqual(ce, c embedings.Embedding) {
	GinkgoHelper()

	// Compare the content
	Expect(ce.Content).To(Equal(c.Content), fmt.Sprintf("Content of Embedding %s not equal", c.ID))

	// Compare the attributes
	for _, a1 := range ce.Attributes {
		found := false

		for _, a2 := range c.Attributes {
			found = false
			if a1.Name == a2.Name && a1.Type == a2.Type {
				found = true
				Expect(a1.Compare(a2)).To(BeTrue(), fmt.Sprintf(
					"%s: attribute '%s' does not match. Expected: %v, Got: %v", c.ID, a1.Name, a1, a2))
				break
			}
		}
		Expect(found).To(BeTrue(), fmt.Sprintf("Attribute %s not found in Embedding '%s'", a1.Name, c.ID))
	}
	if len(ce.Attributes) > 0 {
		Expect(len(ce.Attributes)).To(Equal(len(c.Attributes)),
			fmt.Sprintf("ExpectEmbeddingAreEqual: %s attributes count not equal:\n"+
				"  Expected.Attributes: %#v,\n"+
				"  Got.Attributes:      %#v",
				c.ID, ce.Attributes, c.Attributes))

	}

	// Compare the attributes2
	for _, a1 := range ce.Attributes2 {
		found := false

		for _, a2 := range c.Attributes2 {
			found = false
			if a1.Name == a2.Name && a1.Type == a2.Type {
				found = true
				Expect(a1.Compare(a2)).To(BeTrue(), fmt.Sprintf(
					"%s: attribute2 '%s' does not match. Expected: %v, Got: %v", c.ID, a1.Name, a1, a2))
				break
			}
		}
		Expect(found).To(BeTrue(), fmt.Sprintf("Attribute2 %s not found in Embedding '%s'", a1.Name, c.ID))
	}
	if len(ce.Attributes2) > 0 {
		Expect(len(ce.Attributes2)).To(Equal(len(c.Attributes2)),
			fmt.Sprintf("ExpectEmbeddingAreEqual: %s attributes2 count not equal:\n"+
				"  Expected.Attributes2: %#v,\n"+
				"  Got.Attributes2:      %#v",
				c.ID, ce.Attributes2, c.Attributes2))
	}
}

func ExpectSortedJSONEquals(a, b []byte, param string) {
	GinkgoHelper()

	j1, j2, err := GetSortedJSON(a, b, param)
	Expect(err).To(BeNil())

	By(fmt.Sprintf("Expect that json bytes equal:\n -> json1: %s\n -> json2: %s", prettyJSON(j1), prettyJSON(j2)))

	Expect(SortedJSONEquals(j1, j2)).To(BeTrue(), fmt.Sprintf("JSONs are not equal:\n%s", cmp.Diff(j1, j2)))
}

func ExpectJSONEquals(a, b []byte) {
	GinkgoHelper()

	By("Expect that json bytes equal")
	var d1, d2 map[string]interface{}
	err := json.Unmarshal(a, &d1)
	Expect(err).To(BeNil())
	err = json.Unmarshal(b, &d2)
	Expect(err).To(BeNil())

	Expect(JSONEquals(d1, d2)).To(BeTrue(), fmt.Sprintf("JSONs are not equal:\n%s", cmp.Diff(d1, d2)))
}

func GetSortedJSON(a, b []byte, param string) ([]map[string]interface{}, []map[string]interface{}, error) {
	var d1, d2 []map[string]interface{}
	if err := json.Unmarshal(a, &d1); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(b, &d2); err != nil {
		return nil, nil, err
	}

	sort.Slice(d1, func(i, j int) bool {
		return d1[i][param].(string) < d1[j][param].(string)
	})
	sort.Slice(d2, func(i, j int) bool {
		return d2[i][param].(string) < d2[j][param].(string)
	})

	return d1, d2, nil
}

func SortedJSONEquals(a, b []map[string]interface{}) bool {
	return reflect.DeepEqual(a, b)
}

func JSONEquals(a, b map[string]interface{}) bool {
	return reflect.DeepEqual(a, b)
}

func prettyJSON(body []map[string]interface{}) string {
	b, err := json.Marshal(body)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func RemovedIsolationFromEmbeddingQueue(ctx context.Context, db *pgxpool.Pool, isolationID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Removing isolation %s from vector_store.embedding_queue", isolationID))
	query := `DELETE FROM vector_store.embedding_queue WHERE (content->'iso_id')::jsonb ? $1`
	_, err := db.Exec(ctx, query, isolationID)
	Expect(err).To(BeNil())

}

func ExpectEmbeddingInQueue(ctx context.Context, db *pgxpool.Pool, isoID, colID, docID, embID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect %s/%s/%s/%s is in vector_store.embedding_queue", isoID, colID, docID, embID))
	isInQueue := isEmbeddingInQueue(ctx, db, isoID, colID, docID, embID)
	Expect(isInQueue).To(BeTrue())
}

func ExpectEmbeddingNotInQueue(ctx context.Context, db *pgxpool.Pool, isoID, colID, docID, embID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect %s/%s/%s/%s is not in vector_store.embedding_queue", isoID, colID, docID, embID))
	isInQueue := isEmbeddingInQueue(ctx, db, isoID, colID, docID, embID)
	Expect(isInQueue).To(BeTrue())
}

func isEmbeddingInQueue(ctx context.Context, db *pgxpool.Pool, isoID, colID, docID, embID string) bool {
	By(fmt.Sprintf("-> check if embedding %s/%s/%s/%s is in vector_store.embedding_queue", isoID, colID, docID, embID))

	sqlQuery := `
		SELECT COUNT(*) FROM vector_store.embedding_queue
						 WHERE (content->'iso_id')::jsonb ? $1
						   AND (content->'col_id')::jsonb ? $2
						   AND (content->'doc_id')::jsonb ? $3
						   AND (content->'emb_id')::jsonb ? $4 `

	rows, err := db.Query(ctx, sqlQuery, isoID, colID, docID, embID)
	Expect(err).To(BeNil())
	defer rows.Close()

	var count int
	if rows.Next() {
		if err = rows.Scan(&count); err != nil {
			Expect(err).To(BeNil())
		}
	}
	return count > 0
}

func ExpectDocumentInQueue(ctx context.Context, db *pgxpool.Pool, isoID, colID, docID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect %s/%s/%s is in vector_store.embedding_queue", isoID, colID, docID))
	isInQueue := isDocumentInQueue(ctx, db, isoID, colID, docID)
	Expect(isInQueue).To(BeTrue())
}

func ExpectDocumentNotInQueue(ctx context.Context, db *pgxpool.Pool, isoID, colID, docID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect %s/%s/%s is not in vector_store.embedding_queue", isoID, colID, docID))
	isInQueue := isDocumentInQueue(ctx, db, isoID, colID, docID)
	Expect(isInQueue).To(BeFalse())
}

func isDocumentInQueue(ctx context.Context, db *pgxpool.Pool, isoID, colID, docID string) bool {
	By(fmt.Sprintf("-> check if embedding %s/%s/%s is in vector_store.embedding_queue", isoID, colID, docID))

	sqlQuery := `
		SELECT COUNT(*) FROM vector_store.embedding_queue
						 WHERE (content->'iso_id')::jsonb ? $1
						   AND (content->'col_id')::jsonb ? $2
						   AND (content->'doc_id')::jsonb ? $3 `

	rows, err := db.Query(ctx, sqlQuery, isoID, colID, docID)
	Expect(err).To(BeNil())
	defer rows.Close()

	var count int
	if rows.Next() {
		if err = rows.Scan(&count); err != nil {
			Expect(err).To(BeNil())
		}
	}
	return count > 0
}

func ExpectExecute(ctx context.Context, db *pgxpool.Pool, sqlQuery string, sqlParams ...interface{}) {
	GinkgoHelper()

	_, err := db.Exec(ctx, sqlQuery, sqlParams...)
	Expect(err).To(BeNil())
}

func ExpectTableExists(ctx context.Context, db *pgxpool.Pool, tableName string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect table %s exists in DB", tableName))
	schema, table := helpers.SplitTableName(tableName)
	sqlQuery := fmt.Sprintf("SELECT EXISTS ( SELECT 1 FROM pg_tables WHERE schemaname = '%s' AND tablename = '%s' );", schema, table)
	var exists bool
	err := db.QueryRow(ctx, sqlQuery).Scan(&exists)
	Expect(err).To(BeNil())
	Expect(exists).To(Equal(true))
}

func ExpectTableDoesNotExist(ctx context.Context, db *pgxpool.Pool, tableName string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect table %s does not exist in DB", tableName))
	schema, table := helpers.SplitTableName(tableName)
	sqlQuery := fmt.Sprintf("SELECT EXISTS ( SELECT 1 FROM pg_tables WHERE schemaname = '%s' AND tablename = '%s' );", schema, table)
	var exists bool
	err := db.QueryRow(ctx, sqlQuery).Scan(&exists)
	Expect(err).To(BeNil())
	Expect(exists).To(Equal(false))
}

func ExpectTriggerExists(ctx context.Context, db *pgxpool.Pool, triggerName, tableName string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect trigger %s exists on table %s", triggerName, tableName))
	schema, table := helpers.SplitTableName(tableName)
	sqlQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM information_schema.triggers WHERE trigger_name = '%s'  AND event_object_schema = '%s' AND event_object_table = '%s')", triggerName, schema, table)
	var exists bool
	err := db.QueryRow(ctx, sqlQuery).Scan(&exists)
	Expect(err).To(BeNil())
	Expect(exists).To(Equal(true))
}

func ExpectTriggerDoesNotExist(ctx context.Context, db *pgxpool.Pool, triggerName, tableName string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect trigger %s does not exist on table %s", triggerName, tableName))
	schema, table := helpers.SplitTableName(tableName)
	sqlQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM information_schema.triggers WHERE trigger_name = '%s' AND event_object_schema = '%s' AND event_object_table = '%s')", triggerName, schema, table)
	var exists bool
	err := db.QueryRow(ctx, sqlQuery).Scan(&exists)
	Expect(err).To(BeNil())
	Expect(exists).To(Equal(false))
}

func ExpectNoTablesForIsolation(ctx context.Context, db *pgxpool.Pool, isolationID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect there are no tables for isolation %s ", isolationID))
	sqlQueryTpl := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema  = current_schema AND ( table_name LIKE '%s_%%' OR table_name LIKE '%s_%%')"
	sqlQuery := fmt.Sprintf(sqlQueryTpl, isolationID, isolationID)
	var count int
	err := db.QueryRow(ctx, sqlQuery).Scan(&count)
	Expect(err).To(BeNil())
	Expect(count).To(Equal(0))
}

func ExpectNoIdleTransactionsLeft(ctx context.Context, db *pgxpool.Pool, userName string) {
	GinkgoHelper()

	Eventually(func() []pgStatActivity {
		sqlQuery := `SELECT state, query FROM pg_stat_activity WHERE state LIKE 'idle in transaction%' AND usename=$1`
		rows, err := db.Query(ctx, sqlQuery, userName)
		Expect(err).To(BeNil())
		defer rows.Close()

		var activities []pgStatActivity
		for rows.Next() {
			var dbAct pgStatActivity
			err = rows.Scan(&dbAct.State, &dbAct.Query)
			Expect(err).To(BeNil())
			activities = append(activities, dbAct)
		}
		return activities
	}, "10s").Should(BeNil())
}

func ExpectAttributesGroupExistsInDB(ctx context.Context, db *pgxpool.Pool, isolationID, attrGroupID, description string, attributes []string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect attributes group '%s' exists in DB", attrGroupID))

	attrGrTable := db2.GetTableSmartAttrGroup(isolationID)
	query := fmt.Sprintf("SELECT description, array_to_string(attributes, ',') FROM %s WHERE group_id=$1", attrGrTable)

	rows, err := db.Query(ctx, query, attrGroupID)
	Expect(err).To(BeNil())
	defer rows.Close()

	var dbDescr string
	var dbGrAttr string
	for rows.Next() {
		err = rows.Scan(&dbDescr, &dbGrAttr)
		Expect(err).To(BeNil())
	}
	dbGrAttrs := strings.Split(dbGrAttr, ",")

	Expect(dbDescr).To(Equal(description))
	Expect(len(dbGrAttrs)).To(Equal(len(attributes)))
	for _, attr := range attributes {
		Expect(dbGrAttrs).To(ContainElement(attr))
	}
}

func ExpectAttributesGroupDoesNotExistInDB(ctx context.Context, db *pgxpool.Pool, isolationID, attrGroupID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect attributes group '%s' exists in DB", attrGroupID))

	attrGrTable := db2.GetTableSmartAttrGroup(isolationID)
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE group_id=$1", attrGrTable)

	rows, err := db.Query(ctx, query, attrGroupID)
	Expect(err).To(BeNil())
	defer rows.Close()

	if rows.Next() {
		var count string
		err = rows.Scan(&count)
		Expect(err).To(BeNil())
		Expect(count).To(Equal("0"))
	}
}

type pgStatActivity struct {
	State string
	Query string
}

func ExpectChunksInDatabase(ctx context.Context, dbPool *pgxpool.Pool, isolationID, collectionID, docID string, expectedChunks []embedings.Chunk) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect chunks in database %s/%s/%s", isolationID, collectionID, docID))

	tableEmb := db2.GetTableEmb(isolationID, collectionID)
	tableAttr := db2.GetTableAttr(isolationID, collectionID)

	query := fmt.Sprintf(`
			SELECT emb_id, content,  vector_store.attributes_as_jsonb_by_ids( '%[2]s' , attr_ids2) as attributes
			FROM %[1]s WHERE doc_id=$1 ORDER BY emb_id
			`, tableEmb, tableAttr)

	rows, err := dbPool.Query(ctx, query, docID)
	Expect(err).To(BeNil())
	defer rows.Close()

	dbChunks := make([]embedings.Chunk, 0)
	for i := 0; rows.Next(); i++ {
		ch := embedings.Chunk{}
		err = rows.Scan(&ch.ID, &ch.Content, &ch.Attributes)
		Expect(err).To(BeNil())
		dbChunks = append(dbChunks, ch)

	}
	Expect(len(dbChunks) > 0).To(BeTrue(), "No embeddings found in database for document %s", docID)

	// Check if expected chunks are equal to the ones in the database
	for _, expCh := range expectedChunks {
		found := false
		for _, dbCh := range dbChunks {
			if expCh.ID == dbCh.ID {
				found = true
				Expect(dbCh.Content).To(Equal(expCh.Content))
				if len(expCh.Attributes) > 0 {
					Expect(len(dbCh.Attributes)).To(Equal(len(expCh.Attributes)),
						fmt.Sprintf(
							"ExpectChunksInDatabase: %s attributes count not equal  :\n"+
								"  Expected.Attributes: %#v,\n"+
								"  Got.Attributes:      %#v",
							dbCh.ID, expCh.Attributes, dbCh.Attributes))
					expectAttributesIncluded(expCh.Attributes, dbCh.Attributes)
				}
			}
		}
		Expect(found).To(BeTrue(), fmt.Sprintf("Chunk '%s' not found in database", expCh.ID))
	}
}

// ExpectChunkHasNoAttributesLinked asserts that the embedding row for the given
// emb_id has no attribute IDs linked (attr_ids2 is NULL or empty array).
// Used to verify that attributes with a filtered-out kind were not stored.
func ExpectChunkHasNoAttributesLinked(ctx context.Context, db *pgxpool.Pool,
	isolationID, collectionID, embID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect chunk '%s' in %s/%s has no attributes linked via attr_ids2", embID, isolationID, collectionID))
	tableEmb := db2.GetTableEmb(isolationID, collectionID)
	query := fmt.Sprintf(
		`SELECT COALESCE(array_length(attr_ids2, 1), 0) FROM %s WHERE emb_id = $1`,
		tableEmb)
	var count int
	err := db.QueryRow(ctx, query, embID).Scan(&count)
	Expect(err).To(BeNil())
	Expect(count).To(Equal(0),
		fmt.Sprintf("Expected no attributes linked to embedding '%s', but found %d", embID, count))
}

func expectAttributesIncluded(allAttrs []attributes.Attribute, expectedAttrs []attributes.Attribute) {
	// check if all attributes from expectedAttrs are in allAttrs
	for _, attr := range expectedAttrs {
		found := false
		for _, attr1 := range allAttrs {
			if attr.Name == attr1.Name && attr.Type == attr1.Type {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), fmt.Sprintf(
			"Attribute '%s' not found in expected doc attributes: %v", attr.Name, allAttrs))
	}
}

func ExpectDocumentInDatabase(ctx context.Context, dbPool *pgxpool.Pool, isolationID, collectionID, docID string, docAttrs []attributes.Attribute) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect chunks in database %s/%s/%s", isolationID, collectionID, docID))

	docTableName := db2.GetTableDoc(isolationID, collectionID)
	tableAttrName := db2.GetTableAttr(isolationID, collectionID)

	query := fmt.Sprintf(`
		SELECT vector_store.attributes_as_jsonb_by_ids( '%[2]s', attr_ids) as attributes 
		FROM %[1]s WHERE doc_id=$1
		`, docTableName, tableAttrName)

	rows, err := dbPool.Query(ctx, query, docID)
	Expect(err).To(BeNil())
	defer rows.Close()

	foundRows := false
	for i := 0; rows.Next(); i++ {
		foundRows = true
		var attrs attributes.Attributes
		err = rows.Scan(&attrs)
		Expect(err).To(BeNil())

		// check if the attributes are the same
		for _, chAttr := range attrs {
			Expect(ContainsAttribute(docAttrs, chAttr)).To(BeTrue(),
				"Attribute '%s' not found in expected doc attributes: %v", chAttr.Name, docAttrs)
		}
	}
	Expect(foundRows).To(BeTrue(), "No records found in database for document %s", docID)
}

func ExpectDocumentFoundByFilter(baseURI, isolationID, collectionID, docID string, docFilter []attributes.AttributeFilter) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect document '%s' can by found by filter %v", docID, docFilter))

	endpointURI := fmt.Sprintf("%s/v1/%s/collections/%s/documents", baseURI, isolationID, collectionID)

	reqBodyBytes, err := json.Marshal(docFilter)
	Expect(err).To(BeNil())

	resp, body, err := HttpCall("POST", endpointURI, nil, string(reqBodyBytes))
	Expect(err).To(BeNil())
	Expect(body).NotTo(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var docResp []documents.GetDocumentResponse
	err = json.Unmarshal(body, &docResp)
	Expect(err).To(BeNil())

	found := false
	for _, d := range docResp {
		if d.ID == docID {
			found = true
			break
		}
	}
	Expect(found).To(BeTrue(), fmt.Sprintf("Document %s not found in the database by filter: %v", docID, docFilter))
}

func ExpectCollectionExistsInDB(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect collection '%s' exists in isolation '%s'", collectionID, isolationID))
	ExpectIsolationExistsInDB(ctx, db, isolationID)
	var count int
	query := fmt.Sprintf("SELECT count(*) FROM %s.collections WHERE col_id=$1", db2.GetSchema(isolationID))
	err := db.QueryRow(ctx, query, collectionID).Scan(&count)
	Expect(err).To(BeNil())
	Expect(count > 0).To(BeTrue())
}

func ExpectIndexExists(ctx context.Context, db *pgxpool.Pool, schemaName, tableName, indexName string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect index '%s' exists on '%s.%s' DB", indexName, schemaName, tableName))
	query := "SELECT EXISTS ( SELECT 1 FROM pg_indexes WHERE schemaname = $1 AND tablename = $2 AND indexname = $3 )"
	var exists bool
	err := db.QueryRow(ctx, query, schemaName, tableName, indexName).Scan(&exists)
	Expect(err).To(BeNil())
	Expect(exists).To(Equal(true))
}

func substituteIsolationID(uri string, newIsolationID string) string {
	parsedURL, err := url.Parse(uri)
	Expect(err).To(BeNil())
	pathParts := strings.Split(parsedURL.Path, "/")
	if len(pathParts) > 3 {
		pathParts[2] = newIsolationID
	}
	parsedURL.Path = strings.Join(pathParts, "/")
	return parsedURL.String()
}

func substituteCollectionName(uri string, newIsolationID string) string {
	parsedURL, err := url.Parse(uri)
	Expect(err).To(BeNil())
	pathParts := strings.Split(parsedURL.Path, "/")
	if len(pathParts) > 5 {
		pathParts[4] = newIsolationID
	}
	parsedURL.Path = strings.Join(pathParts, "/")
	return parsedURL.String()
}

func ExpectServiceReturns404IfIsolationDoesNotExist(method, uri string) {
	GinkgoHelper()

	isoID := fmt.Sprintf("iso-not-exist-%s", RandStringRunes(5))
	isoID = strings.ToLower(isoID)
	newUri := substituteIsolationID(uri, isoID)
	By(fmt.Sprintf("Expect service returns 404 if isolation does not exist when calling: [%s] %s", method, newUri))
	resp, body, err := HttpCall(method, newUri, nil, "{}")
	Expect(err).To(BeNil())
	Expect(body).NotTo(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	expectedBody := fmt.Sprintf(`{"code":"404","message":"isolation '%s' not found. Please install GenAIVectorStoreIsolation SCE first"}`, isoID)
	Expect(body).To(Equal([]byte(expectedBody)))
}

func ExpectServiceReturns404IfCollectionDoesNotExists(method, uri string) {
	GinkgoHelper()

	colID := fmt.Sprintf("col-not-exist-%s", RandStringRunes(5))
	colID = strings.ToLower(colID)
	newUri := substituteCollectionName(uri, colID)
	By(fmt.Sprintf("Expect service returns 404 if isolation does not exist when calling: [%s] %s", method, newUri))
	resp, body, err := HttpCall(method, newUri, nil, "{}")
	Expect(err).To(BeNil())
	Expect(body).NotTo(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	expectedBody := fmt.Sprintf(`{"code":"404","message":"collection '%s' not found. Please insert data before retrieving"}`, colID)
	Expect(body).To(Equal([]byte(expectedBody)))
}

func ExpectEmbeddingMetadataInDatabase(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, embID, metadataKey, metadataValue string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect embedding metadata in database %s/%s/%s key=%s", isolationID, collectionID, embID, metadataKey))
	tableEmbMeta := db2.GetTableEmbMeta(isolationID, collectionID)
	sqlQuery := fmt.Sprintf(`SELECT metadata_value  FROM %[1]s  WHERE emb_id=$1 and metadata_key=$2`, tableEmbMeta, embID)
	rows, err := db.Query(ctx, sqlQuery, embID, metadataKey)
	Expect(err).To(BeNil())
	defer rows.Close()

	var value string
	if rows.Next() {
		err = rows.Scan(&value)
		Expect(err).To(BeNil())
	}
	Expect(value).To(Equal(metadataValue), fmt.Sprintf("Metadata value for emb_id '%s' and key '%s' not found in %s", embID, metadataKey, tableEmbMeta))
}

func ExpectDocumentMetadataInDatabase(ctx context.Context, db *pgxpool.Pool, isolationID, collectionID, docID, metadataKey, metadataValue string) {
	GinkgoHelper()

	By(fmt.Sprintf("Expect document metadata in database %s/%s/%s key=%s", isolationID, collectionID, docID, metadataKey))
	tableDocMeta := db2.GetTableDocMeta(isolationID, collectionID)
	sqlQuery := fmt.Sprintf(`SELECT metadata_value  FROM %[1]s  WHERE doc_id=$1 and metadata_key=$2`, tableDocMeta, docID)
	rows, err := db.Query(ctx, sqlQuery, docID, metadataKey)
	Expect(err).To(BeNil())
	defer rows.Close()

	var value string
	if rows.Next() {
		err = rows.Scan(&value)
		Expect(err).To(BeNil())
	}
	Expect(value).To(Equal(metadataValue), fmt.Sprintf("Metadata value for doc_id '%s' and key '%s' not found in %s", docID, metadataKey, tableDocMeta))
}

// ExpectAttributesMigrated verifies attributes were migrated to JSONB columns
func ExpectAttributesMigrated(ctx context.Context, database *pgxpool.Pool, isolationID, collectionID, documentID string) {
	GinkgoHelper()

	By(fmt.Sprintf("Verifying attributes were migrated for %s/%s/%s", isolationID, collectionID, documentID))

	tableDoc := db2.GetTableDoc(isolationID, collectionID)
	query := fmt.Sprintf(`SELECT doc_attributes FROM %s WHERE doc_id = $1`, tableDoc)

	var docAttrs []byte
	err := database.QueryRow(ctx, query, documentID).Scan(&docAttrs)
	Expect(err).To(BeNil(), "Failed to query document")

	Expect(docAttrs).NotTo(BeNil(), "doc_attributes should not be NULL after migration")
	Expect(len(docAttrs)).To(BeNumerically(">", 2), "doc_attributes should contain data (more than just '{}')")
}

// ExpectSchemaVersion verifies the schema version
func ExpectSchemaVersion(ctx context.Context, database interface{}, expectedVersion string) {
	GinkgoHelper()

	db := database.(*pgxpool.Pool)
	By(fmt.Sprintf("Expecting schema version to be '%s'", expectedVersion))

	query := `SELECT value FROM vector_store.configuration WHERE key = 'schema_version'`
	var version string
	err := db.QueryRow(ctx, query).Scan(&version)
	Expect(err).To(BeNil(), "Failed to get schema version")
	Expect(version).To(Equal(expectedVersion))
}

// GetSchemaVersion retrieves the current schema version
func GetSchemaVersion(ctx context.Context, database interface{}) (string, error) {
	db := database.(*pgxpool.Pool)
	query := `SELECT value FROM vector_store.configuration WHERE key = 'schema_version'`
	var version string
	err := db.QueryRow(ctx, query).Scan(&version)
	return version, err
}

// ExpectColumnExists verifies a column exists in a table
func ExpectColumnExists(ctx context.Context, database interface{}, schemaName, tableName, columnName string) {
	GinkgoHelper()

	db := database.(*pgxpool.Pool)
	By(fmt.Sprintf("Expecting column '%s' exists in '%s.%s'", columnName, schemaName, tableName))

	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
		)`
	var exists bool
	err := db.QueryRow(ctx, query, schemaName, tableName, columnName).Scan(&exists)
	Expect(err).To(BeNil(), "Failed to check column existence")
	Expect(exists).To(BeTrue(), fmt.Sprintf("Column '%s' should exist in '%s.%s'", columnName, schemaName, tableName))
}

// ExpectColumnDoesNotExist verifies a column does not exist in a table
func ExpectColumnDoesNotExist(ctx context.Context, database interface{}, schemaName, tableName, columnName string) {
	GinkgoHelper()

	db := database.(*pgxpool.Pool)
	By(fmt.Sprintf("Expecting column '%s' does not exist in '%s.%s'", columnName, schemaName, tableName))

	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
		)`
	var exists bool
	err := db.QueryRow(ctx, query, schemaName, tableName, columnName).Scan(&exists)
	Expect(err).To(BeNil(), "Failed to check column existence")
	Expect(exists).To(BeFalse(), fmt.Sprintf("Column '%s' should not exist in '%s.%s'", columnName, schemaName, tableName))
}

// CloseDatabase closes the database connection pool
func CloseDatabase(database interface{}) {
	if db, ok := database.(*pgxpool.Pool); ok {
		db.Close()
	}
}

// ExpectDocAttributesContain verifies doc_attributes JSONB contains specific key-value pair
func ExpectDocAttributesContain(ctx context.Context, database interface{}, isolationID, collectionID, docID, attrName, expectedValue string) {
	GinkgoHelper()

	db := database.(*pgxpool.Pool)
	By(fmt.Sprintf("Expecting doc_attributes contains '%s'='%s' for %s/%s/%s", attrName, expectedValue, isolationID, collectionID, docID))

	tableDoc := db2.GetTableDoc(isolationID, collectionID)
	query := fmt.Sprintf(`
		SELECT doc_attributes->$2->'values'->>0 AS value
		FROM %s 
		WHERE doc_id = $1 AND doc_attributes ? $2
	`, tableDoc)

	var actualValue string
	err := db.QueryRow(ctx, query, docID, attrName).Scan(&actualValue)
	Expect(err).To(BeNil(), fmt.Sprintf("Failed to query doc_attributes for attribute '%s'", attrName))
	Expect(actualValue).To(Equal(expectedValue), fmt.Sprintf("Attribute '%s' should have value '%s'", attrName, expectedValue))
}

// ExpectEmbAttributesContain verifies emb_attributes JSONB contains specific key-value pair
func ExpectEmbAttributesContain(ctx context.Context, database interface{}, isolationID, collectionID, embID, attrName, expectedValue string) {
	GinkgoHelper()

	db := database.(*pgxpool.Pool)
	By(fmt.Sprintf("Expecting emb_attributes contains '%s'='%s' for %s/%s/%s", attrName, expectedValue, isolationID, collectionID, embID))

	tableEmb := db2.GetTableEmb(isolationID, collectionID)
	query := fmt.Sprintf(`
		SELECT emb_attributes->$2->'values'->>0 AS value
		FROM %s 
		WHERE emb_id = $1 AND emb_attributes ? $2
	`, tableEmb)

	var actualValue string
	err := db.QueryRow(ctx, query, embID, attrName).Scan(&actualValue)
	Expect(err).To(BeNil(), fmt.Sprintf("Failed to query emb_attributes for attribute '%s'", attrName))
	Expect(actualValue).To(Equal(expectedValue), fmt.Sprintf("Attribute '%s' should have value '%s'", attrName, expectedValue))
}

// ExpectCombinedAttributesContain verifies combined attributes JSONB contains specific key-value pair
func ExpectCombinedAttributesContain(ctx context.Context, database interface{}, isolationID, collectionID, embID, attrName, expectedValue string) {
	GinkgoHelper()

	db := database.(*pgxpool.Pool)
	By(fmt.Sprintf("Expecting combined attributes contains '%s'='%s' for %s/%s/%s", attrName, expectedValue, isolationID, collectionID, embID))

	tableEmb := db2.GetTableEmb(isolationID, collectionID)
	query := fmt.Sprintf(`
		SELECT attributes->$2->'values'->>0 AS value
		FROM %s 
		WHERE emb_id = $1 AND attributes ? $2
	`, tableEmb)

	var actualValue string
	err := db.QueryRow(ctx, query, embID, attrName).Scan(&actualValue)
	Expect(err).To(BeNil(), fmt.Sprintf("Failed to query combined attributes for attribute '%s'", attrName))
	Expect(actualValue).To(Equal(expectedValue), fmt.Sprintf("Attribute '%s' should have value '%s'", attrName, expectedValue))
}

// GetConfigurationCount returns the count of configuration entries matching a pattern
func GetConfigurationCount(ctx context.Context, database interface{}, keyPattern string) int {
	db := database.(*pgxpool.Pool)
	query := `SELECT COUNT(*) FROM vector_store.configuration WHERE key LIKE $1`
	var count int
	err := db.QueryRow(ctx, query, keyPattern).Scan(&count)
	Expect(err).To(BeNil(), "Failed to get configuration count")
	return count
}

// ExpectAllDocAttributesMigrated verifies that all documents in a table have their attributes migrated
// (i.e., doc_attributes column is not NULL and not empty). This function polls the database and waits
// for the migration to complete, making it suitable for use after triggering an asynchronous migration.
func ExpectAllDocAttributesMigrated(ctx context.Context, database interface{}, schemaName, tableName string, timeout time.Duration) {
	db := database.(*pgxpool.Pool)
	By(fmt.Sprintf("Verifying ALL documents in %s.%s have attributes migrated", schemaName, tableName))

	fullTableName := fmt.Sprintf("%s.%s", schemaName, tableName)

	Eventually(func() error {
		// Count total documents
		var totalDocs int
		countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, fullTableName)
		err := db.QueryRow(ctx, countQuery).Scan(&totalDocs)
		if err != nil {
			return fmt.Errorf("failed to count total documents: %w", err)
		}
		if totalDocs == 0 {
			return fmt.Errorf("no documents found in table")
		}

		// Count documents with NULL doc_attributes
		var nullCount int
		nullQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE doc_attributes IS NULL`, fullTableName)
		err = db.QueryRow(ctx, nullQuery).Scan(&nullCount)
		if err != nil {
			return fmt.Errorf("failed to count NULL doc_attributes: %w", err)
		}

		// Count documents with empty doc_attributes (just '{}')
		var emptyCount int
		emptyQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE doc_attributes = '{}'::jsonb`, fullTableName)
		err = db.QueryRow(ctx, emptyQuery).Scan(&emptyCount)
		if err != nil {
			return fmt.Errorf("failed to count empty doc_attributes: %w", err)
		}

		// All documents should have migrated attributes
		if nullCount > 0 {
			return fmt.Errorf("migration incomplete: %d out of %d documents have NULL doc_attributes", nullCount, totalDocs)
		}
		if emptyCount > 0 {
			return fmt.Errorf("migration incomplete: %d out of %d documents have empty doc_attributes", emptyCount, totalDocs)
		}

		return nil
	}, timeout, 500*time.Millisecond).Should(Succeed(), "All documents must have non-NULL, non-empty doc_attributes after migration")
}

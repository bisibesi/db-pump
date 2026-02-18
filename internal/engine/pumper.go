package engine

import (
	"database/sql"
	"db-pump/internal/dialect"
	"db-pump/internal/schema"
	"fmt"
	"strings"
	"time"
)

// getDataTypeMaxValue returns the maximum value for a given data type
func getDataTypeMaxValue(dataType string) int {
	switch strings.ToLower(dataType) {
	case "tinyint":
		return 255
	case "smallint":
		return 32767
	case "mediumint":
		return 8388607
	case "int", "integer":
		return 2147483647
	default:
		return 2147483647 // Default to int max
	}
}

// calculateMaxInsertCount calculates the maximum number of rows that can be inserted
// based on IDENTITY column data type constraints
func calculateMaxInsertCount(table *schema.Table, requestedCount int) int {
	maxCount := requestedCount

	// Check for IDENTITY columns with limited data types
	for _, c := range table.Columns {
		if c.IsAutoInc {
			typeMax := getDataTypeMaxValue(c.DataType)
			if typeMax < maxCount {
				maxCount = typeMax
				fmt.Printf("[LIMIT] Table %s: IDENTITY column %s (%s) limits max rows to %d\n",
					table.Name, c.Name, c.DataType, typeMax)
			}
		}
	}

	return maxCount
}

func Pump(db *sql.DB, d dialect.Dialect, tables []*schema.Table, count int, onProgress func()) ([]schema.PumpResult, error) {
	var results []schema.PumpResult
	fkPool := make(map[string][]interface{})

	for _, table := range tables {
		// 기존 데이터 건수 확인
		var initialCount int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table.Name)).Scan(&initialCount)

		// 데이터 타입 제약에 따른 최대 삽입 건수 계산
		adjustedCount := calculateMaxInsertCount(table, count)

		// Check for identity column
		hasIdentity := false
		for _, c := range table.Columns {
			if c.IsAutoInc {
				hasIdentity = true
				break
			}
		}

		// UI 진행바와 겹치지 않게 내부적으로만 처리
		tx, _ := db.Begin()
		if err := d.BeforeTable(tx, table.Name, hasIdentity); err != nil {
			fmt.Printf("Warning: BeforeTable hook failed for %s: %v\n", table.Name, err)
		}

		var insertCols []*schema.Column
		var colNames []string
		for _, c := range table.Columns {
			if !c.IsAutoInc {
				insertCols = append(insertCols, c)
				colNames = append(colNames, c.Name)
			}
		}

		query := d.InsertQuery(table.Name, colNames)
		inserted := 0
		attempts := 0

		// Track used combinations for composite PK tables
		usedCombinations := make(map[string]bool)
		hasCompositePK := false
		pkCount := 0
		for _, c := range table.Columns {
			if c.IsPK {
				pkCount++
			}
		}
		if pkCount > 1 {
			hasCompositePK = true
		}

		// Track used values for UNIQUE columns
		usedUniqueValues := make(map[string]map[interface{}]bool)
		for _, c := range insertCols {
			if c.IsUnique {
				usedUniqueValues[c.Name] = make(map[interface{}]bool)
			}
		}

		// 목표치 채우기 로직 (중복 시 재시도)
		// adjustedCount를 사용하여 데이터 타입 제약 준수
		for inserted < adjustedCount && attempts < adjustedCount*10 {
			attempts++
			// Use attempt number for sequential FK selection in composite PK tables
			values, ok := generateRowWithIndex(table, insertCols, fkPool, attempts)
			if !ok {
				// FK constraint cannot be satisfied - skip this table
				break
			}

			// Check for composite PK duplicates
			if hasCompositePK {
				// Build combination key from PK values
				var pkValues []string
				for i, c := range insertCols {
					if c.IsPK {
						pkValues = append(pkValues, fmt.Sprintf("%v", values[i]))
					}
				}
				combinationKey := strings.Join(pkValues, "|")
				if usedCombinations[combinationKey] {
					// Skip this duplicate combination
					continue
				}
				usedCombinations[combinationKey] = true
			}

			// Check for UNIQUE column duplicates
			skipRow := false
			for i, c := range insertCols {
				if c.IsUnique {
					if usedUniqueValues[c.Name][values[i]] {
						// Skip this row - UNIQUE value already used
						skipRow = true
						break
					}
				}
			}
			if skipRow {
				continue
			}

			// Mark UNIQUE values as used
			for i, c := range insertCols {
				if c.IsUnique {
					usedUniqueValues[c.Name][values[i]] = true
				}
			}

			_, err := tx.Exec(query, values...)
			if err == nil {
				inserted++
				if onProgress != nil {
					onProgress()
				}
			} else if attempts <= 3 {
				// Log first 3 errors
				fmt.Printf("[DEBUG] Table %s attempt %d: %v\nQuery: %s\n", table.Name, attempts, err, query)
			}
		}

		d.AfterTable(tx, table.Name, hasIdentity) // SET IDENTITY_INSERT OFF
		tx.Commit()

		// 실제 들어간 개수 확인 (Verification)
		var finalCount int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table.Name)).Scan(&finalCount)
		actual := finalCount - initialCount

		status := "OK"
		var errMsg string
		if actual < adjustedCount {
			status = "MISSING DATA"
			// Try to capture reasoning (not perfect but helpful)
			if inserted == 0 && attempts > 0 {
				errMsg = "Failed to insert any rows. Check logs for details."
			} else if inserted < adjustedCount {
				errMsg = fmt.Sprintf("Only inserted %d out of %d. High failure rate?", actual, adjustedCount)
			}
		}

		results = append(results, schema.PumpResult{
			TableName: table.Name,
			Target:    count, // 원래 요청한 건수 표시
			Actual:    actual,
			Status:    status,
			ErrorMsg:  errMsg,
		})

		// FK 풀 갱신 (다음 자식 테이블을 위해)
		updateFKPool(db, table, fkPool)
	}

	return results, nil
}

func generateRow(table *schema.Table, cols []*schema.Column, fkPool map[string][]interface{}) ([]interface{}, bool) {
	return generateRowWithIndex(table, cols, fkPool, 0)
}

func generateRowWithIndex(table *schema.Table, cols []*schema.Column, fkPool map[string][]interface{}, index int) ([]interface{}, bool) {
	var values []interface{}
	for _, col := range cols {
		val, ok := getSmartValWithIndex(col, table, fkPool, index)
		if !ok {
			// FK constraint cannot be satisfied
			return nil, false
		}
		values = append(values, val)
	}
	return values, true
}

func updateFKPool(db *sql.DB, table *schema.Table, fkPool map[string][]interface{}) {
	var pk string
	for _, c := range table.Columns {
		if c.IsPK {
			pk = c.Name
			break
		}
	}
	if pk == "" {
		return
	}

	// PK 값 수집 (MSSQL/Postgres 호환)
	query := fmt.Sprintf("SELECT %s FROM %s", pk, table.Name)
	rows, err := db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id interface{}
		if err := rows.Scan(&id); err == nil {
			fkPool[table.Name] = append(fkPool[table.Name], id)
		}
	}
}

// VerifyInjection checks the actual row counts after pumping and returns results.
func VerifyInjection(db *sql.DB, results []schema.PumpResult) []schema.PumpResult {
	var verifiedResults []schema.PumpResult
	for _, res := range results {
		var currentCount int
		// Check current count again
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", res.TableName)).Scan(&currentCount)

		status := "OK"
		if err != nil {
			status = fmt.Sprintf("VERIFY_FAIL: %v", err)
		} else if currentCount < res.Target {
			status = fmt.Sprintf("PARTIAL: %d/%d", currentCount, res.Target)
		}

		verifiedResults = append(verifiedResults, schema.PumpResult{
			TableName: res.TableName,
			Target:    res.Target,
			Actual:    currentCount,
			Status:    status,
			ErrorMsg:  res.ErrorMsg,
		})
	}
	return verifiedResults
}

func getSmartVal(col *schema.Column, t *schema.Table, pool map[string][]interface{}) (interface{}, bool) {
	return getSmartValWithIndex(col, t, pool, 0)
}

func getSmartValWithIndex(col *schema.Column, t *schema.Table, pool map[string][]interface{}, index int) (interface{}, bool) {
	for _, fk := range t.ForeignKeys {
		if fk.Column == col.Name {
			if vals, ok := pool[fk.RefTable]; ok && len(vals) > 0 {
				// For UNIQUE FK columns, always use sequential selection to avoid duplicates
				if col.IsUnique || index > 0 {
					return vals[index%len(vals)], true
				}
				return vals[time.Now().UnixNano()%int64(len(vals))], true
			}
			// FK pool is empty - likely circular dependency
			// If nullable, return NULL
			if col.IsNullable {
				return nil, true
			}
			// For UNIQUE FK columns, use index-based value to avoid duplicates
			if col.IsUnique && index > 0 {
				// Use index as the FK value (assumes referenced table has sequential IDs)
				return index, true
			}
			// Use default value 1 (assumes the referenced table will have ID 1)
			// This helps with circular dependencies like staff <-> store
			// Improved specific fallback for tinyint/byte IDs like store_id
			if strings.Contains(strings.ToLower(col.DataType), "tinyint") {
				return 1, true
			}
			return 1, true
		}
	}
	return GenerateValue(col, t.Name), true
}

package schema

import (
	"database/sql"
	"db-pump/internal/dialect"
	"fmt"
	"strings"
)

// analyzeMeaning is now in meaning.go

// ---------------------------------------------------------------------
// 2. Schema Analysis Logic
// ---------------------------------------------------------------------

func Analyze(db *sql.DB, d dialect.Dialect, schemaName string) ([]*Table, error) {
	// [Interface-First]: Delegate schema resolution to the dialect
	target := d.GetSchemaName(schemaName)

	// Use map for O(1) lookups, with normalized keys for case-insensitive matching (Oracle support)
	tableMap := make(map[string]*Table)
	var tables []*Table

	// --- Step 1: Fetch Tables ---
	// [Error Handling]: Return error to allow rollback/handling by caller
	rows, err := db.Query(d.GetTablesQuery(target), target)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		t := &Table{Name: name, Dependencies: []string{}}
		// Store with normalized key (UPPERCASE) for robust lookups
		tableMap[strings.ToUpper(name)] = t
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tables: %w", err)
	}

	// --- Step 2: Fetch Columns ---
	colRows, err := db.Query(d.GetColumnsQuery(target), target)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer colRows.Close()

	for colRows.Next() {
		var tName, cName, dType, cType, isNull, cKey, extra, isUnique, comment sql.NullString
		var cLen sql.NullString // Use String for safety

		if err := colRows.Scan(&tName, &cName, &dType, &cType, &cLen, &isNull, &cKey, &extra, &isUnique, &comment); err != nil {
			// Log warning but continue? Or fail? Better to fail if schema is inconsistent.
			// But for resilience, we might skip. However, "Atomic" implies all or nothing.
			return nil, fmt.Errorf("failed to scan column (table: %s): %w", tName.String, err)
		}

		if !tName.Valid || !cName.Valid {
			continue // Skip invalid rows
		}

		// Lookup using Normalized Key
		if t, ok := tableMap[strings.ToUpper(tName.String)]; ok {
			// PK Detection
			isPK := strings.Contains(cKey.String, "PRI") || strings.Contains(cKey.String, "PRIMARY")

			// AutoInc Detection
			isAutoInc := false
			if extra.Valid {
				extraLower := strings.ToLower(extra.String)
				isAutoInc = strings.Contains(extraLower, "auto_increment") ||
					strings.Contains(extraLower, "identity") ||
					strings.Contains(extraLower, "nextval")
			}

			// Unique Detection
			isUniqueCol := false
			if isUnique.Valid {
				isUniqueCol = strings.Contains(isUnique.String, "UNIQUE")
			}

			// Meaning Analysis
			meaning := AnalyzeMeaning(cName.String, comment.String)

			col := &Column{
				Name:       cName.String,
				DataType:   d.NormalizeType(dType.String), // Use raw type if cType complex expression failed
				IsNullable: isNull.String == "YES",
				IsPK:       isPK,
				IsAutoInc:  isAutoInc,
				IsUnique:   isUniqueCol,
				Comment:    comment.String,
				Meaning:    meaning,
			}

			// Handle Length safely
			if cLen.Valid && cLen.String != "" {
				var length int
				if _, err := fmt.Sscanf(cLen.String, "%d", &length); err == nil {
					col.Length = length
				} else {
					var fLength float64
					if _, err := fmt.Sscanf(cLen.String, "%f", &fLength); err == nil {
						col.Length = int(fLength)
					}
				}
			}
			t.Columns = append(t.Columns, col)
		}
	}
	if err := colRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating columns: %w", err)
	}

	// --- Step 3: Fetch Foreign Keys ---
	fkRows, err := db.Query(d.GetForeignKeysQuery(target), target)
	if err != nil {
		// FK query might fail on some DBs if permissions are missing.
		// We return error to be safe.
		return nil, fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer fkRows.Close()

	for fkRows.Next() {
		var tName, cConst, cName, rTable, rCol sql.NullString
		if err := fkRows.Scan(&tName, &cConst, &cName, &rTable, &rCol); err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		if tName.Valid && rTable.Valid && tName.String != rTable.String {
			// Lookup using Normalized Key
			tKey := strings.ToUpper(tName.String)
			rKey := strings.ToUpper(rTable.String)

			if t, ok := tableMap[tKey]; ok {
				// Verify if referenced table exists in our map (to avoid external refs we can't handle)
				if _, exists := tableMap[rKey]; exists {
					// Add dependency only if it's a known table
					actualRTableName := tableMap[rKey].Name // Get original case name
					t.Dependencies = append(t.Dependencies, actualRTableName)
					t.ForeignKeys = append(t.ForeignKeys, &ForeignKey{
						Column:    cName.String,
						RefTable:  actualRTableName,
						RefColumn: rCol.String,
					})
				}
			}
		}
	}
	if err := fkRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating foreign keys: %w", err)
	}

	return SortTablesByFKCount(tables), nil
}

// ---------------------------------------------------------------------
// 3. Sorting Algorithm (Topological / Greedy)
// ---------------------------------------------------------------------

// SortTablesByFKCount sorts tables by dependency order.
// It handles circular dependencies by using a scoring system.
func SortTablesByFKCount(tables []*Table) []*Table {
	var sorted []*Table
	processed := make(map[string]bool)

	// Keep looping until all tables are processed
	for len(sorted) < len(tables) {
		added := false

		// Pass 1: Add tables whose dependencies are fully satisfied
		for _, t := range tables {
			if processed[t.Name] {
				continue
			}

			allDepsProcessed := true
			for _, depName := range t.Dependencies {
				if !processed[depName] {
					allDepsProcessed = false
					break
				}
			}

			if allDepsProcessed {
				sorted = append(sorted, t)
				processed[t.Name] = true
				added = true
			}
		}

		// Pass 2: If no table added, we have a cycle. Break it using heuristic score.
		if !added {
			var bestTable *Table
			bestScore := -999999

			for _, t := range tables {
				if processed[t.Name] {
					continue
				}

				// Score calculation:
				// Base: 0
				// Penalty: Number of Unprocessed FKs (Prefer fewer dependencies)
				// Bonus: Participation in Cycle (Prefer breaking cycles early)
				score := 0

				// Count failing dependencies
				unprocessedDeps := 0
				for _, dep := range t.Dependencies {
					if !processed[dep] {
						unprocessedDeps++
					}
				}
				score -= (unprocessedDeps * 100)

				// Cycle Detection Bonus
				// If this table is referenced by one of its own dependencies (already processed or not),
				// it's a strong candidate to break.
				// However, simplified cycle heuristic:
				// If I depend on A, and A is not processed.
				// Check if A depends on Me.
				isCircular := false
				for _, depName := range t.Dependencies {
					if !processed[depName] {
						// Find depTable object
						for _, cand := range tables {
							if cand.Name == depName {
								// Check if cand depends on t.Name
								for _, candDep := range cand.Dependencies {
									if candDep == t.Name {
										isCircular = true
										break
									}
								}
								break
							}
						}
					}
					if isCircular {
						break
					}
				}

				if isCircular {
					score += 500 // Priority boost
				}

				// Tie-breaker: Name (Deterministic)
				if score > bestScore {
					bestScore = score
					bestTable = t
				} else if score == bestScore {
					if bestTable == nil || t.Name > bestTable.Name { // Alphabetical reverse preference? Or standard?
						bestTable = t
					}
				}
			}

			if bestTable != nil {
				sorted = append(sorted, bestTable)
				processed[bestTable.Name] = true
				fmt.Printf("[Sort] Breaking circular dependency: %s (Score: %d)\n", bestTable.Name, bestScore)
			} else {
				// Should not happen if tables > sorted
				fmt.Println("[Sort] Error: Deadlock in sorting logic? Remaining tables cannot be sorted.")
				break
			}
		}
	}

	return sorted
}

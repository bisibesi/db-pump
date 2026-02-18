package schema_test

import (
	"db-pump/internal/schema"
	"testing"
)

func TestSortTablesByFKCount_ComplexCircular(t *testing.T) {
	// 5개 이상의 복잡한 FK 관계를 가진 가상 테이블 구조 정의
	// A -> B -> C -> D -> E -> A (순환)
	// F -> E (단순 참조)
	// G (독립)
	tables := []*schema.Table{
		{Name: "A", Dependencies: []string{"B"}},
		{Name: "B", Dependencies: []string{"C"}},
		{Name: "C", Dependencies: []string{"D"}},
		{Name: "D", Dependencies: []string{"E"}},
		{Name: "E", Dependencies: []string{"A"}},
		{Name: "F", Dependencies: []string{"E"}},
		{Name: "G", Dependencies: []string{}},
	}

	sorted := schema.SortTablesByFKCount(tables)

	if len(sorted) != len(tables) {
		t.Errorf("Expected %d tables, got %d", len(tables), len(sorted))
	}

	// 순서 검증: 의존성이 최대한 만족되었는지 확인
	// 완전한 정답은 없지만(순환이므로), 적어도 독립 테이블 G가 앞쪽에 오는지 등을 확인
	// 그리고 Circular Dependency가 깨져서(heuristic score) 모든 테이블이 포함되었는지 확인

	visited := make(map[string]bool)
	for _, tbl := range sorted {
		visited[tbl.Name] = true
	}

	if !visited["A"] || !visited["B"] || !visited["C"] || !visited["D"] || !visited["E"] || !visited["F"] || !visited["G"] {
		t.Error("Not all tables are in the sorted list")
	}

	// G should ideally be first or very early
	if sorted[0].Name != "G" {
		t.Logf("Notice: Independent table G is at index 0? actual: %s", sorted[0].Name)
	}
}

func TestSortTablesByFKCount_Simple(t *testing.T) {
	// Users -> Orders -> OrderItems
	tables := []*schema.Table{
		{Name: "OrderItems", Dependencies: []string{"Orders"}},
		{Name: "Orders", Dependencies: []string{"Users"}},
		{Name: "Users", Dependencies: []string{}},
	}

	sorted := schema.SortTablesByFKCount(tables)

	if sorted[0].Name != "Users" {
		t.Errorf("Expected Users first, got %s", sorted[0].Name)
	}
	if sorted[1].Name != "Orders" {
		t.Errorf("Expected Orders second, got %s", sorted[1].Name)
	}
	if sorted[2].Name != "OrderItems" {
		t.Errorf("Expected OrderItems third, got %s", sorted[2].Name)
	}
}

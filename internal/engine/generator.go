package engine

import (
	"db-pump/internal/schema"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// 한국어 데이터 상수 (이름/주소/전화번호용)
// EngToKorMap moved to dicts.go

var engKeys []string

func init() {
	for k := range EngToKorMap {
		engKeys = append(engKeys, k)
	}
}

// 1. 영문 텍스트 생성 (사전에 있는 단어 위주로)
func generateEnglishText(wordCount int) string {
	var words []string
	for i := 0; i < wordCount; i++ {
		words = append(words, engKeys[seededRand.Intn(len(engKeys))])
	}
	return strings.Join(words, " ")
}

// 2. 번역 함수 (영 -> 한)
func translateToKorean(text string) string {
	var result []string
	words := strings.Split(text, " ")
	for _, w := range words {
		if val, ok := EngToKorMap[w]; ok {
			result = append(result, val)
		} else {
			result = append(result, w) // 사전에 없으면 그대로
		}
	}
	return strings.Join(result, " ")
}

// 한국어 고유 데이터 생성 함수들
func GenerateKoreanName() string {
	return LastNames[seededRand.Intn(len(LastNames))] + FirstNames[seededRand.Intn(len(FirstNames))]
}

func GenerateKoreanAddress() string {
	city := Cities[seededRand.Intn(len(Cities))]
	district := Districts[seededRand.Intn(len(Districts))]
	street := Streets[seededRand.Intn(len(Streets))]
	number := seededRand.Intn(100) + 1
	return fmt.Sprintf("%s %s %s %d번길", city, district, street, number)
}

func GenerateKoreanPhone() string {
	return fmt.Sprintf("010-%04d-%04d", seededRand.Intn(10000), seededRand.Intn(10000))
}

func truncate(s string, limit int) string {
	if limit <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return s
}

// GenerateValue generates a random value based on column definition
func GenerateValue(col *schema.Column, tableName string) interface{} {
	dataType := strings.ToLower(col.DataType)
	colName := strings.ToLower(col.Name)
	meaning := col.Meaning

	// 0. ENUM / CHECK 처리
	if strings.Contains(colName, "special_features") || strings.Contains(colName, "features") ||
		strings.Contains(colName, "rating") {
		if col.IsNullable {
			return nil
		}
		return ""
	}
	if len(col.EnumValues) > 0 {
		return col.EnumValues[seededRand.Intn(len(col.EnumValues))]
	}

	// 1. 문자열 타입 처리 (Meaning 분석을 최우선 적용)
	if strings.Contains(dataType, "char") || strings.Contains(dataType, "text") ||
		strings.Contains(dataType, "varchar") || strings.Contains(dataType, "string") ||
		strings.Contains(dataType, "year") {

		isID := strings.HasSuffix(colName, "id") || strings.HasSuffix(colName, "_id")

		// Meaning 기반 생성
		if strings.Contains(meaning, "year") || strings.Contains(colName, "year") {
			// year는 ID 여부 상관없이 값(연도) 생성
			return fmt.Sprintf("%d", 2000+seededRand.Intn(26))
		}
		if !isID && (strings.Contains(meaning, "phone") || strings.Contains(colName, "phone")) {
			return truncate(GenerateKoreanPhone(), col.Length)
		}
		if !isID && (strings.Contains(meaning, "email") || strings.Contains(colName, "email")) {
			return truncate(gofakeit.Email(), col.Length)
		}
		if !isID && (strings.Contains(meaning, "name") || strings.Contains(colName, "name") ||
			strings.Contains(colName, "first") || strings.Contains(colName, "last")) {
			if col.Length > 0 && col.Length < 3 {
				// 짧은 이름 (성만)
				return truncate(string([]rune(LastNames[seededRand.Intn(len(LastNames))])), col.Length)
			}
			return truncate(GenerateKoreanName(), col.Length)
		}
		if !isID && (strings.Contains(meaning, "address") || strings.Contains(colName, "address")) {
			if strings.Contains(colName, "2") {
				return truncate(fmt.Sprintf("%d층 %d호", seededRand.Intn(20)+1, seededRand.Intn(10)+1), col.Length)
			}
			return truncate(GenerateKoreanAddress(), col.Length)
		}
		if strings.Contains(meaning, "zipcode") || strings.Contains(colName, "zip") || strings.Contains(colName, "postal") {
			return fmt.Sprintf("%05d", seededRand.Intn(100000))
		}
		if strings.Contains(meaning, "yesno") || strings.Contains(colName, "active") || strings.Contains(colName, "is_") {
			// 문자열 'Y'/'N' 생성
			if seededRand.Intn(2) == 0 {
				return "Y"
			}
			return "N"
		}
		if !isID && (strings.Contains(meaning, "title") || strings.Contains(meaning, "subject")) {
			eng := generateEnglishText(2)
			kor := translateToKorean(eng)
			return truncate(kor, col.Length)
		}
		if !isID && (strings.Contains(meaning, "description") || strings.Contains(meaning, "content") ||
			strings.Contains(meaning, "comment") || strings.Contains(meaning, "text")) {
			eng := generateEnglishText(10)
			kor := translateToKorean(eng)
			return truncate(kor, col.Length)
		}
		if !isID && (strings.Contains(meaning, "country") || strings.Contains(colName, "country")) {
			return "대한민국"
		}
		if !isID && (strings.Contains(meaning, "city") || strings.Contains(colName, "city")) {
			return truncate(Cities[seededRand.Intn(len(Cities))], col.Length)
		}
		if !isID && (strings.Contains(meaning, "district") || strings.Contains(colName, "district")) {
			return truncate(Districts[seededRand.Intn(len(Districts))], col.Length)
		}

		// (Meaning 미발견 시) 일반 텍스트 데이터 생성

		// Language/Category (테이블명 의존)
		if tableName == "language" || tableName == "category" {
			eng := generateEnglishText(1)
			kor := translateToKorean(eng)
			return truncate(fmt.Sprintf("%s-%d", kor, seededRand.Intn(1000)), col.Length)
		}

		// 기본 텍스트
		if col.Length > 0 && col.Length < 20 {
			eng := generateEnglishText(1)
			kor := translateToKorean(eng)
			return truncate(kor, col.Length)
		}
		eng := generateEnglishText(5)
		kor := translateToKorean(eng)
		return truncate(kor, col.Length)
	}

	// 2. 숫자, 날짜 등 나머지 타입 처리 (Meaning 무시하고 타입 위주로 생성)

	// 2.1 날짜/시간 타입 (주의: MSSQL 호환성을 위해 포맷팅된 문자열 반환)
	if strings.Contains(dataType, "date") || strings.Contains(dataType, "time") {
		// PostgreSQL Partitioned Table Support (payment_pYYYY_MM)
		if strings.HasPrefix(tableName, "payment_p") {
			parts := strings.Split(tableName, "_p")
			if len(parts) == 2 {
				dateParts := strings.Split(parts[1], "_")
				if len(dateParts) >= 2 {
					year, err1 := strconv.Atoi(dateParts[0])
					month, err2 := strconv.Atoi(dateParts[1])
					if err1 == nil && err2 == nil {
						start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
						// 해당 월의 마지막 날 (다음달 1일 - 1초)
						end := start.AddDate(0, 1, 0).Add(-time.Second)
						val := gofakeit.DateRange(start, end)
						return val.Format("2006-01-02 15:04:05")
					}
				}
			}
		}

		val := gofakeit.DateRange(time.Now().AddDate(-1, 0, 0), time.Now())
		if dataType == "date" { // 정확히 date인 경우
			return val.Format("2006-01-02")
		}
		if dataType == "time" { // 정확히 time인 경우
			return val.Format("15:04:05")
		}
		// datetime, timestamp 등
		return val.Format("2006-01-02 15:04:05")
	}

	// 2.2 숫자 타입
	if strings.Contains(dataType, "int") || strings.Contains(dataType, "integer") {
		// Boolean-like column handling
		if strings.Contains(colName, "active") || strings.Contains(colName, "enabled") ||
			strings.Contains(meaning, "yesno") || strings.Contains(colName, "is_") {
			return seededRand.Intn(2) // 0 or 1
		}

		if strings.Contains(dataType, "tinyint") {
			return gofakeit.Number(0, 127) // Safe range for signed/unsigned logic simplicity
		}
		if strings.Contains(dataType, "smallint") {
			return gofakeit.Number(1, 30000)
		}
		// year 컬럼이 int일 경우 여기서 처리
		if strings.Contains(colName, "year") || strings.Contains(meaning, "year") {
			return 2000 + seededRand.Intn(26)
		}

		// Respect column length (precision) if available
		maxVal := 50000
		if col.Length > 0 && col.Length < 10 { // Only apply for reasonable small precisions
			limit := 1
			for i := 0; i < col.Length; i++ {
				limit *= 10
			}
			limit -= 1
			if limit < maxVal {
				maxVal = limit
				if maxVal < 1 {
					maxVal = 9 // Minimum fallback
				}
			}
		}
		return gofakeit.Number(1, maxVal)
	}

	if strings.Contains(dataType, "decimal") || strings.Contains(dataType, "numeric") ||
		strings.Contains(dataType, "float") || strings.Contains(dataType, "double") {
		return gofakeit.Price(0.99, 99.99)
	}

	// 2.3 불린 타입
	if strings.Contains(dataType, "bool") || strings.Contains(dataType, "bit") {
		return gofakeit.Bool()
	}

	// PostgreSQL tsvector 타입 처리
	if strings.Contains(dataType, "tsvector") {
		return generateEnglishText(5)
	}

	// 2.4 바이너리 타입
	if strings.Contains(dataType, "binary") || strings.Contains(dataType, "blob") || strings.Contains(dataType, "bytea") {
		return []byte("dummy")
	}

	return nil
}

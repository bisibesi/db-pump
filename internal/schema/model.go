package schema

type Table struct {
	Name         string
	Columns      []*Column
	ForeignKeys  []*ForeignKey
	Dependencies []string // 의존성 분석용
}

type Column struct {
	Name       string
	DataType   string
	Length     int
	IsNullable bool
	IsPK       bool
	IsAutoInc  bool
	IsUnique   bool
	EnumValues []string
	Comment    string // DB 스키마 코멘트 (MS_Description 등)
	Meaning    string // 약어 또는 코멘트 분석을 통해 파악된 의미 (예: "phone", "email")
}

type ForeignKey struct {
	Column    string
	RefTable  string
	RefColumn string
}

// 리포트용 구조체
type PumpResult struct {
	TableName string
	Target    int
	Actual    int
	Status    string
	ErrorMsg  string
}

package schema

import "strings"

var abbreviations = map[string]string{
	// Common Nouns
	"nm": "name", "dt": "date", "no": "number", "cd": "code",
	"desc": "description", "amt": "amount", "cnt": "count", "qty": "quantity",
	"addr": "address", "tel": "phone", "hp": "phone", "ph": "phone",
	"biz": "business", "pwd": "password", "passwd": "password", "pw": "password",
	"img": "image", "file": "file", "path": "path", "url": "url",
	"ip": "ip", "zip": "zipcode", "post": "zipcode",
	"msg": "message", "txt": "text", "tit": "title", "subj": "subject",
	"doc": "document", "usr": "user", "emp": "employee",
	"dept": "department", "grp": "group", "cat": "category",
	"loc": "location", "lat": "latitude", "lng": "longitude", "lon": "longitude",
	"geo": "geometry", "st": "street", "prov": "province", "dist": "district",
	"bal": "balance", "calc": "calculation", "rst": "result", "rslt": "result",
	"std": "standard", "avg": "average", "mid": "id", "uid": "id", "pid": "id",

	// Verbs / Status
	"reg": "registered", "mod": "modified", "del": "deleted", "cre": "created",
	"upd": "updated", "yn": "yesno", "stat": "status", "sts": "status",
	"typ": "type", "kind": "kind", "val": "value",
	"ord": "order", "seq": "sequence", "idx": "index",
	"bg": "background", "fg": "foreground",
	"brd": "board", "art": "article", "auth": "authority",
	"is": "yesno", "use": "yesno", "flg": "flag",
}

func AnalyzeMeaning(colName, comment string) string {
	c := strings.ToLower(comment)
	n := strings.ToLower(colName)

	// 1. Priority based on comment keywords (Korean/English)
	if strings.Contains(c, "전화") || strings.Contains(c, "휴대폰") || strings.Contains(c, "연락처") ||
		strings.Contains(c, "핸드폰") || strings.Contains(c, "mobile") || strings.Contains(c, "phone") {
		return "phone"
	}
	if strings.Contains(c, "이메일") || strings.Contains(c, "메일") || strings.Contains(c, "email") || strings.Contains(c, "mail") {
		return "email"
	}
	if strings.Contains(c, "주소") || strings.Contains(c, "거주지") || strings.Contains(c, "address") {
		return "address"
	}
	if strings.Contains(c, "우편") || strings.Contains(c, "zip") || strings.Contains(c, "postal") {
		return "zipcode"
	}
	if strings.Contains(c, "이름") || strings.Contains(c, "성명") || strings.Contains(c, "name") {
		return "name"
	}
	if strings.Contains(c, "아이디") || strings.Contains(c, "user_id") {
		return "id"
	}
	if strings.Contains(c, "비밀번호") || strings.Contains(c, "패스워드") || strings.Contains(c, "암호") || strings.Contains(c, "password") {
		return "password"
	}
	if strings.Contains(c, "제목") || strings.Contains(c, "타이틀") {
		return "title"
	}
	if strings.Contains(c, "내용") || strings.Contains(c, "설명") || strings.Contains(c, "desc") {
		return "description"
	}
	if strings.Contains(c, "날짜") || strings.Contains(c, "일시") || strings.Contains(c, "date") || strings.Contains(c, "time") {
		return "date"
	}
	if strings.Contains(c, "금액") || strings.Contains(c, "가격") || strings.Contains(c, "단가") || strings.Contains(c, "price") || strings.Contains(c, "cost") {
		return "price"
	}
	if strings.Contains(c, "수량") || strings.Contains(c, "개수") || strings.Contains(c, "count") || strings.Contains(c, "qty") {
		return "count"
	}
	if strings.Contains(c, "여부") || strings.Contains(c, "flag") || strings.Contains(c, "yn") {
		return "yesno"
	}
	if strings.Contains(c, "국가") || strings.Contains(c, "나라") || strings.Contains(c, "country") {
		return "country"
	}
	if strings.Contains(c, "도시") || strings.Contains(c, "city") {
		return "city"
	}
	if strings.Contains(c, "ip") {
		return "ip"
	}

	// 2. Abbreviation Analysis from Column Name
	parts := strings.Split(n, "_")
	var decodedParts []string
	for _, part := range parts {
		if full, ok := abbreviations[part]; ok {
			decodedParts = append(decodedParts, full)
		} else {
			decodedParts = append(decodedParts, part)
		}
	}

	return strings.Join(decodedParts, " ")
}

# ⛽ DB Pump (한국어 가이드)

DB Pump는 개발 및 테스트 목적으로 현실적인 더미 데이터를 생성해 데이터베이스에 채워주는 강력한 도구입니다.

데이터베이스 스키마를 지능적으로 분석하여 테이블 관계(Foreign Key)를 이해하고, 복잡한 의존성 구조와 순환 참조를 자동으로 처리하며, 단순한 임의 문자열이 아닌 컬럼명과 주석을 기반으로 의미 있는 데이터(이름, 주소, 이메일 등)를 생성합니다.

---

## 🚀 주요 특징 (Key Features)

*   **다중 데이터베이스 지원**: **MySQL**, **PostgreSQL**, **MSSQL (SQL Server)**, **Oracle** 데이터베이스를 완벽하게 지원합니다.
*   **스마트 스키마 분석**: 테이블, 컬럼, 기본 키(PK), 외래 키(FK)를 자동으로 감지합니다.
*   **의존성 해결**: FK 의존성에 따라 데이터 삽입 순서를 자동으로 정렬하며, 순환 참조(Circular Reference) 문제도 우회하여 처리합니다.
*   **의미 기반 데이터 생성**: 컬럼 이름(예: `nm`, `addr`)이나 주석을 분석하여 적절한 형식(이름, 주소 등)의 데이터를 생성합니다.
*   **한국어 데이터 지원**: 설정을 통해 한국어 이름, 주소 등을 생성할 수 있습니다.
*   **유연한 테이블 필터링**: 설정 파일이나 CLI 명령어로 특정 테이블만 선택하여 데이터를 생성할 수 있습니다.
*   **고성능**: 대용량 데이터 삽입을 위한 트랜잭션 및 배치 처리에 최적화되어 있습니다.

---

## 📦 설치 (Installation)

소스 코드를 빌드하여 실행 파일을 생성할 수 있습니다.

```bash
# 저장소 복제
git clone https://github.com/your-repo/db-pump.git
cd db-pump

# 빌드
go build -o db-pump.exe main.go
```

---

## ⚙️ 설정 (`db-pump.yaml`)

데이터베이스 연결 정보와 기본 설정은 `db-pump.yaml` 파일에서 관리합니다.

```yaml
databases:
  - name: "Local MySQL"
    driver: "mysql"
    dsn: "root:root@tcp(127.0.0.1:3306)/sakila?parseTime=true"
    active: true  # 이 연결을 사용하려면 true로 설정

  - name: "Local PostgreSQL"
    driver: "postgres"
    dsn: "postgres://user:password@localhost:5432/dbname?sslmode=disable"
    active: false

settings:
  default_count: 1000       # 테이블당 기본 생성 데이터 수
  language: "ko"            # 데이터 생성 언어 (예: "ko" - 한국어)
  tables: []                # 데이터 생성 대상 테이블 리스트 (비어있으면 전체 테이블)
                            # 예시: ["users", "orders"]
```

---

## 🛠️ 사용법 (Usage)

### 1. 기본 실행 (전체 테이블 채우기)

활성화된 데이터베이스의 모든 테이블에 설정된 기본 개수만큼 데이터를 생성합니다.

```bash
# Linux / macOS
./db-pump fill

# Windows
db-pump.exe fill
```

### 2. 생성 개수 지정

각 테이블당 생성할 데이터의 개수를 직접 지정합니다.

```bash
# Linux / macOS
./db-pump fill --count 500

# Windows
db-pump.exe fill --count 500
```

### 3. 특정 테이블만 실행

원하는 테이블만 선택하여 데이터를 생성합니다. (설정 파일의 `tables` 값을 덮어씁니다.)

```bash
# 'actor'와 'city' 테이블만 처리
# Linux / macOS
./db-pump fill --tables "actor,city"

# Windows
db-pump.exe fill --tables "actor,city"
```

### 4. 기존 데이터 삭제 후 실행 (Clean)

데이터를 넣기 전에 테이블을 비웁니다. **주의: 기존 데이터가 모두 삭제됩니다.**

```bash
# Linux / macOS
./db-pump fill --clean

# Windows
db-pump.exe fill --clean
```

### 5. 모의 실행 (Dry Run)

데이터베이스에 실제로 쓰지 않고, 실행 순서와 스키마 분석 결과만 확인합니다.

```bash
# Linux / macOS
./db-pump fill --dry-run

# Windows
db-pump.exe fill --dry-run
```

### 6. CLI 전용 모드 (설정 파일 없음)

`db-pump.yaml` 파일 없이 플래그를 통해 직접 연결 정보를 입력하여 실행할 수 있습니다.

```bash
```bash
# MySQL
# DSN 형식: user:password@tcp(host:port)/dbname
# JDBC 예시: jdbc:mysql://host:port/dbname
./db-pump fill --dsn "root:password@tcp(localhost:3306)/dbname" --driver mysql

# PostgreSQL
# DSN 형식: postgres://user:password@host:port/dbname?sslmode=disable
# JDBC 예시: jdbc:postgresql://host:port/dbname
./db-pump fill --dsn "postgres://user:password@localhost:5432/dbname?sslmode=disable" --driver postgres

# MSSQL (SQL Server)
# DSN 형식: sqlserver://user:password@host:port?database=dbname
# JDBC 예시: jdbc:sqlserver://host:port;databaseName=dbname
./db-pump fill --dsn "sqlserver://sa:password@localhost:1433?database=dbname" --driver sqlserver

# Oracle
# DSN 형식: oracle://user:password@host:port/service_name
# JDBC 예시: jdbc:oracle:thin:@host:port:service_name
./db-pump fill --dsn "oracle://user:password@localhost:1521/service" --driver oracle
```

---

## 📝 지원 데이터베이스 & 드라이버

| 데이터베이스 | 드라이버 이름 | 비고 |
| :--- | :--- | :--- |
| **MySQL** | `mysql` | `INSERT IGNORE`를 사용하여 중복 키 오류를 무시합니다. |
| **PostgreSQL**| `postgres` | `ON CONFLICT DO NOTHING`을 사용합니다. |
| **MSSQL** | `sqlserver` | `IDENTITY_INSERT` 및 제약 조건(Constraint)을 자동으로 처리합니다. |
| **Oracle** | `oracle` | Oracle Instant Client 또는 호환 환경이 필요합니다. |


## 🤝 기여 (Contributing)

이 프로젝트에 기여하고 싶으시다면 언제든 Pull Request를 보내주세요! 버그 제보나 기능 제안도 환영합니다.

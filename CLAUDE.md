# Loop - Anthropic API Key Management & Proxy Gateway

## Development Rules

### 时间必须使用 UTC 且格式化为字符串写入 SQLite

Go 的 `time.Time` 对象被 `modernc.org/sqlite` 驱动存储为 `time.String()` 格式
（如 `2026-05-27 13:49:23.341308 +0800 CST m=+105.895498543`），
SQLite 的 `date()` / `datetime()` 函数无法解析该格式，会导致时间查询全部失效。

**两条规则：**

1. 所有时间统一使用 UTC：`time.Now().UTC()`
2. 写入 SQLite 时必须格式化为字符串：`t.UTC().Format("2006-01-02 15:04:05")`

repo 包中已有 `fmtTime()` 辅助函数，直接使用即可。

**错误示例：**
```go
// ❌ 本地时间 + 传 time.Time 对象给 SQL
CreatedAt: time.Now()
db.Exec("INSERT INTO t (created_at) VALUES (?)", time.Now())
```

**正确示例：**
```go
// ✅ UTC + 格式化字符串
CreatedAt: time.Now().UTC()
db.Exec("INSERT INTO t (created_at) VALUES (?)", fmtTime(time.Now()))
```

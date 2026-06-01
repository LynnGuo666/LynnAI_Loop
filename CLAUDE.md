# Loop - Anthropic API Key Management & Proxy Gateway

## Development Rules

### 时间必须使用 UTC

所有涉及时间的操作 **必须使用 UTC**，禁止使用本地时间 (`time.Now()`)。

- Go 代码中写入数据库的时间字段统一使用 `time.Now().UTC()`
- SQLite 查询中 `datetime('now')` 本身就是 UTC，因此两者必须匹配
- 禁止混用本地时间和 UTC，否则会导致时间范围查询（timeseries、today stats 等）数据丢失

**错误示例：**
```go
CreatedAt: time.Now()        // ❌ 本地时间，与 SQLite UTC 不一致
```

**正确示例：**
```go
CreatedAt: time.Now().UTC()  // ✅ UTC，与 SQLite datetime('now') 一致
```

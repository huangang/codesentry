# Database Update Scripts

这个目录包含用于批量更新数据库的实用脚本。

## 脚本列表

### 1. update_min_score

更新所有项目的最低评分阈值（min_score）。

**功能**: 批量设置所有项目的最低分数为 75 分

**运行方式**:
```bash
cd backend
go run cmd/scripts/update_min_score/main.go
```

**修改配置**: 编辑 `update_min_score/main.go` 文件中的以下变量来调整分数:
```go
// 在 main 函数中修改这个值
result := db.Model(&Project{}).Where("deleted_at IS NULL").Update("min_score", 75)
```

---

### 2. update_ignore_patterns

更新所有项目的忽略模式（ignore_patterns）。

**功能**: 批量设置所有项目的配置文件忽略模式

**当前配置**: `etc/config.yaml,etc/.env,etc/config.yml`

**运行方式**:
```bash
cd backend
go run cmd/scripts/update_ignore_patterns/main.go
```

**修改配置**: 编辑 `update_ignore_patterns/main.go` 文件中的以下变量:
```go
// 在 main 函数中修改这个值
newIgnorePatterns := "etc/config.yaml,etc/.env,etc/config.yml"
```

---

## 数据库连接配置

所有脚本都使用相同的数据库连接字符串。如需修改，请在各脚本的 `main` 函数中更新 `dsn` 变量：

```go
dsn := "downtown:downtown#2013@tcp(10.11.15.44:3306)/codesentry?charset=utf8mb4&parseTime=True&loc=Local"
```

## 注意事项

1. **备份数据**: 运行这些脚本前，建议先备份数据库
2. **只更新未删除的项目**: 所有脚本都会过滤 `deleted_at IS NULL` 的记录
3. **查看结果**: 脚本会显示更新前后的数据对比
4. **影响范围**: 这些脚本会批量更新所有符合条件的项目

## 脚本输出示例

脚本会显示:
- 数据库连接状态
- 更新前的示例数据
- 受影响的记录总数
- 更新后的示例数据
- 更新成功确认信息

## 目录结构

```
backend/cmd/scripts/
├── README.md                           # 本文档
├── update_min_score/
│   └── main.go                        # 更新最低分数脚本
└── update_ignore_patterns/
    └── main.go                        # 更新忽略模式脚本
```

## 添加新脚本

如需添加新的数据库更新脚本:

1. 在 `backend/cmd/scripts/` 下创建新目录
2. 在该目录中创建 `main.go` 文件
3. 使用 `main` 包名
4. 参考现有脚本的结构编写代码
5. 更新本 README 文档

这样可以避免多个 main 包在同一目录下产生冲突。

# WANF - WJQserver's Aligned Nodal Format

**WANF (WJQserver's Aligned Nodal Format)** 是一种富有表现力的声明式配置语言。其核心设计哲学是：**配置语法应尽可能地反映其映射的数据结构**。

这使得WANF在保持简洁、人类可读的同时，具备了上下文感知的高级特性，能够为开发者提供符合直觉的配置体验。

[![Go Reference](https://pkg.go.dev/badge/wanf.svg)](https://pkg.go.dev/github.com/WJQSERVER/wanf)
[![Go Report Card](https://goreportcard.com/badge/github.com/WJQSERVER/wanf)](https://goreportcard.com/report/github.com/WJQSERVER/wanf)

---

### ✨ 特性亮点

*   **Go语言原生体验**: 语法设计与Go的结构体、切片、映射的声明方式非常相似。原生支持 `time.Duration` 等Go类型。
*   **结构清晰，无歧义**: 使用显式的 `{}` 定义块边界，避免了类似YAML中因缩进而产生的常见错误，使得复杂配置的结构一目了然。
*   **强大的功能**: 支持变量声明 (`var`)、环境变量读取 (`env()`) 和文件导入 (`import`)，让你的配置轻松实现模块化和复用。
*   **注释保留**: `wanflint fmt` 格式化工具被设计为可以完美保留您的所有注释（包括行尾注释和块注释），确保配置文件的可读性和可维护性.
*   **类型安全的映射**: 通过Go结构体标签 `wanf:"..."`，可以实现强大且类型安全的配置到结构体的映射，包括自动将列表转换为map。

### 🚀 快速开始

#### 1. 语法概览

这是一个典型的 `.wanf` 配置文件:

```wanf
// 全局变量，用于复用
var default_timeout = 10s

// 主服务器配置
server "main_api" {
    host = env("API_HOST", "0.0.0.0") // 从环境变量读取，若不存在则使用默认值
    port = 8080

    // 可以在块内继续嵌套
    features {
        rate_limit_enabled = true
        timeout = ${default_timeout} // 引用变量
    }
}

// 数据库配置 (无标签块)
database {
    user = "admin"
    password = env("DB_PASSWORD") // 敏感信息应使用环境变量
}

// 启用的功能列表
enabled_features = [
    "feature_a",
    "feature_b",
]

/*
 这是一个块注释,
 用于描述下面的日志服务
*/
log {
    level = "info"
    path = "/var/log/app.log"
}
```

#### 2. 安装 `wanflint`

`wanflint` 是WANF的官方Linter和格式化工具。

```sh
go install github.com/WJQSERVER/wanf/cmd/wanflint@latest
```

#### 3. 使用 `wanflint`

*   **格式化文件**: 自动整理您的配置文件，并保留注释。

    ```sh
    wanflint fmt your_config.wanf
    ```

*   **检查语法和风格**:

    ```sh
    wanflint lint your_config.wanf
    ```

### Go 语言集成

在您的Go应用中使用WANF非常简单。

#### 1. 定义你的Go结构体

使用 `wanf` 标签来映射配置文件中的键。

```go
package main

import (
    "fmt"
    "github.com/WJQSERVER/wanf"
    "time"
)

type Config struct {
    Server   map[string]ServerConfig `wanf:"server"`
    Database DatabaseConfig        `wanf:"database"`
    Features []string                `wanf:"enabled_features"`
    Log      LogConfig               `wanf:"log"`
}

type ServerConfig struct {
    Host     string        `wanf:"host"`
    Port     int           `wanf:"port"`
    Features FeatureConfig `wanf:"features"`
}

type FeatureConfig struct {
    RateLimitEnabled bool          `wanf:"rate_limit_enabled"`
    Timeout          time.Duration `wanf:"timeout"`
}

type DatabaseConfig struct {
    User     string `wanf:"user"`
    Password string `wanf:"password"`
}

type LogConfig struct {
    Level string `wanf:"level"`
    Path  string `wanf:"path"`
}
```

#### 2. 解析配置文件

```go
func main() {
    var cfg Config

    // 从文件解析
    err := wanf.DecodeFile("your_config.wanf", &cfg)
    if err != nil {
        panic(err)
    }

    // 或者从字节流解析
    // data, _ := os.ReadFile("your_config.wanf")
    // err = wanf.Decode(data, &cfg)

    fmt.Printf("Parsed config: %+v\n", cfg)
    fmt.Printf("Main API timeout: %s\n", cfg.Server["main_api"].Features.Timeout)
}
```

### 贡献

本项目欢迎各种形式的贡献，包括但不限于：
*   Bug报告和修复
*   功能建议
*   文档改进
*   为其他语言（如Python, Rust, TypeScript）实现WANF解析器

在您贡献代码前，请先创建一个Issue来讨论您的想法。

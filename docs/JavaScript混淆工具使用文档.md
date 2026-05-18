# JavaScript混淆工具使用文档

## 概述

本文档介绍hjtpx项目中的JavaScript代码混淆和加密工具的基本使用方法。该工具位于`backend/internal/tools/javascript_obfuscator.go`，提供了多种代码保护和混淆功能。

**功能范围**：变量名混淆、字符串加密、代码压缩、控制流平坦化、反调试检测等。

**已知限制**：
- 混淆后的代码在某些极端情况下可能存在兼容性问题
- 控制流平坦化可能影响代码性能
- 高级混淆功能可能会显著增加代码体积

## 快速开始

### 基本使用

```go
import "github.com/hjtpx/hjtpx/backend/internal/tools"

// 使用默认配置混淆代码
code := `function hello() { return "world"; }`
obfuscated, err := tools.Obfuscate(code)

// 使用自定义配置
config := tools.ObfuscatorConfig{
    EnableVariableObfuscation: true,
    EnableStringEncryption:    true,
    StringEncryptionMethod:    "aes-gcm",
}
obfuscated, err := tools.ObfuscateWithConfig(code, config)
```

### 创建混淆器实例

```go
obfuscator := tools.NewObfuscator()

// 使用自定义配置
config := tools.ObfuscatorConfig{
    EnableVariableObfuscation: true,
    EnableStringEncryption:    true,
    EnableCodeCompression:     true,
}
obfuscator := tools.NewObfuscator(config)
result, err := obfuscator.Obfuscate(code)
```

## 配置选项

### 基础配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `EnableVariableObfuscation` | bool | true | 启用变量名混淆 |
| `EnableStringEncryption` | bool | true | 启用字符串加密 |
| `EnableCodeCompression` | bool | true | 启用代码压缩 |
| `EnableControlFlowFlattening` | bool | true | 启用控制流平坦化 |
| `EnableDeadCodeInjection` | bool | false | 注入死代码 |
| `EnableFunctionWrapping` | bool | true | 函数包装 |
| `StringEncryptionKey` | []byte | "hjtpx-obfuscate-key-2024" | 加密密钥 |
| `CompressWhitespace` | bool | true | 压缩空白字符 |
| `RemoveComments` | bool | true | 移除注释 |
| `PreserveConsole` | bool | true | 保留console对象 |

### 高级配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `EnableAdvancedAntiDebug` | bool | false | 高级反调试 |
| `EnableSelfDestruct` | bool | false | 自毁机制 |
| `EnableMemoryProtection` | bool | false | 内存保护 |
| `EnableCodeVirtualization` | bool | false | 代码虚拟化 |
| `StringEncryptionMethod` | string | "aes-gcm" | 加密算法 |
| `EnableNameMangling` | bool | true | 名称混淆 |
| `EnableScopeTracking` | bool | false | 作用域跟踪 |

## 加密算法

工具支持多种字符串加密算法：

### 1. AES-GCM（默认）

```go
config := tools.ObfuscatorConfig{
    StringEncryptionMethod: "aes-gcm",
}
```

对称加密算法，安全性较高，基本能用。

### 2. RC4

```go
config := tools.ObfuscatorConfig{
    StringEncryptionMethod: "rc4",
}
```

流加密算法，速度较快，安全性一般。

### 3. ChaCha20

```go
config := tools.ObfuscatorConfig{
    StringEncryptionMethod: "chacha20",
}
```

现代流加密算法，性能好，目前看还算稳定。

### 4. XOR

```go
config := tools.ObfuscatorConfig{
    StringEncryptionMethod: "xor",
}
```

简单异或加密，速度最快，但安全性较低，碰巧适合对性能要求高的场景。

## 反调试机制

### 基础反调试

```go
code := tools.InjectAntiDebug(code)
```

检测开发者工具窗口大小变化。

### 高级反调试

```go
code := tools.InjectAdvancedAntiDebug(code)
```

**检测方法**：
- 窗口尺寸异常检测（阈值160px）
- debugger语句时间差检测
- Console API劫持检测
- Firebug检测
- 原型链修改检测

**保护措施**：
- F12键拦截
- Ctrl+Shift+I/J拦截
- 右键菜单禁用
- 定时器持续检测

**注意事项**：
- 高级反调试可能在某些合法使用场景下触发
- 建议仅在核心业务逻辑中使用

## 代码分析工具

### 代码分析

```go
analyzer := tools.AnalyzeCode(code)
metrics := analyzer.GetMetrics()
```

返回指标包括：
- 代码行数
- 函数数量
- 字符串数量
- 变量数量
- 注释数量
- 圈复杂度

### 混淆质量评估

```go
quality := tools.EstimateObfuscationQuality(original, obfuscated)
```

返回指标包括：
- 熵值分析
- 大小比率
- 可读性评分
- 函数复杂度降低

## 混淆等级

### 预设混淆等级

```go
// 轻度混淆（快速）
result := tools.OptimizeObfuscation(code, 1)

// 中度混淆
result := tools.OptimizeObfuscation(code, 2)

// 高度混淆
result := tools.OptimizeObfuscation(code, 3)
```

### 自动判断等级

```go
level := tools.GetObfuscationLevel(code)
```

根据代码复杂度自动判断适合的混淆等级。

## 完整性保护

### 代码签名

```go
signature := tools.GenerateCodeSignature(code, secret)
valid := tools.VerifyCodeSignature(code, signature, secret)
```

### 完整性检查

```go
check := tools.CreateCodeIntegrityModule(code, key)
```

### 哈希值计算

```go
hash := tools.HashCode(code)
valid := tools.VerifyCodeIntegrity(originalHash, code)
```

## 密钥管理

### 生成随机密钥

```go
key, err := tools.GenerateRandomKey(32)
hexKey, err := tools.GenerateHexKey(32)
```

### 密钥验证

```go
valid := tools.ValidateKey(key)
```

## 实际使用示例

### 示例1：基础混淆

```go
package main

import (
    "fmt"
    "github.com/hjtpx/hjtpx/backend/internal/tools"
)

func main() {
    code := `
    function calculateSum(arr) {
        var sum = 0;
        for (var i = 0; i < arr.length; i++) {
            sum += arr[i];
        }
        return sum;
    }
    `

    obfuscator := tools.NewObfuscator(tools.ObfuscatorConfig{
        EnableVariableObfuscation: true,
        EnableStringEncryption:    true,
        EnableCodeCompression:     true,
    })

    result, err := obfuscator.Obfuscate(code)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Original length: %d\n", len(code))
    fmt.Printf("Obfuscated length: %d\n", len(result))
    fmt.Printf("Stats: %+v\n", obfuscator.GetStats())
}
```

### 示例2：高级安全配置

```go
obfuscator := tools.NewObfuscator(tools.ObfuscatorConfig{
    EnableVariableObfuscation:   true,
    EnableStringEncryption:      true,
    EnableCodeCompression:       true,
    EnableControlFlowFlattening: true,
    EnableAdvancedAntiDebug:     true,
    EnableSelfDestruct:          true,
    EnableMemoryProtection:      true,
    StringEncryptionMethod:      "aes-gcm",
    StringEncryptionKey:         []byte("your-secure-key-here"),
})

result, err := obfuscator.ApplyAdvancedObfuscation(code)
```

### 示例3：批量文件混淆

```go
err := tools.ObfuscateFile("input.js", "output.js")
if err != nil {
    fmt.Printf("Failed to obfuscate file: %v\n", err)
}
```

### 示例4：性能评估

```go
code := `// 您的代码`
estimatedTime := tools.EstimateObfuscationTime(len(code))
fmt.Printf("Estimated time: %s\n", estimatedTime)

level := tools.GetObfuscationLevel(code)
fmt.Printf("Recommended level: %d\n", level)

result := tools.OptimizeObfuscation(code, level)
```

## 最佳实践

### 1. 选择合适的混淆等级

```
轻度混淆（等级1）：
  - 适用：性能敏感代码
  - 特性：变量混淆 + 代码压缩

中度混淆（等级2）：
  - 适用：一般业务逻辑
  - 特性：+ 字符串加密 + 函数包装

高度混淆（等级3）：
  - 适用：核心安全逻辑
  - 特性：+ 控制流平坦化 + 死代码注入
```

### 2. 密钥管理建议

**不要**：使用默认密钥用于生产环境

**应该**：
```go
// 生成安全的随机密钥
key, _ := tools.GenerateRandomKey(32)
config := tools.ObfuscatorConfig{
    StringEncryptionKey: key,
}
```

### 3. 测试建议

```go
// 混淆前验证代码
valid, msg := tools.ValidateObfuscatedCode(code)
if !valid {
    fmt.Printf("Invalid code: %s\n", msg)
}

// 评估混淆质量
quality := tools.EstimateObfuscationQuality(original, obfuscated)
fmt.Printf("Obfuscation quality: %.2f%%\n", quality["unreadability_percent"])
```

### 4. 性能考虑

- 大文件混淆可能需要较长时间
- 控制流平坦化会增加代码体积（通常50-100%）
- 建议在构建流程中处理混淆，不要在运行时混淆

## 已知问题和限制

### 1. 代码压缩问题

当前版本的代码压缩功能在某些情况下可能不够彻底，空白字符压缩基本正常，但可能还有优化空间。

### 2. 控制流平坦化

- 可能影响代码可读性，但不影响功能
- 在某些浏览器中可能存在性能开销
- 循环结构处理可能不够完美

### 3. 字符串加密

- 加密后的字符串在运行时会被解密
- 如果密钥泄露，混淆效果会大打折扣
- 不同的加密算法有不同的性能特征

### 4. 反调试机制

- 可能与某些合法的浏览器扩展冲突
- 在某些特殊网络环境下可能误报
- 不可能完全阻止专业逆向工程师

### 5. 兼容性

- 混淆后的代码在现代浏览器中基本能正常运行
- 不保证在所有浏览器和设备上完全一致
- 建议在目标环境进行充分测试

## 调试技巧

### 查看混淆统计

```go
obfuscator := tools.NewObfuscator(config)
obfuscator.Obfuscate(code)
stats := obfuscator.GetStats()
fmt.Printf("Variables obfuscated: %d\n", stats["variables_obfuscated"])
fmt.Printf("Strings encrypted: %d\n", stats["strings_encrypted"])
```

### 生成混淆报告

```go
report := tools.GenerateObfuscationReport(original, obfuscated, config)
fmt.Printf("%+v\n", report)
```

### 生成混淆证书

```go
certificate := tools.GenerateObfuscationCertificate(original, obfuscated, config)
fmt.Println(certificate)
```

## 安全声明

**不是什么**：
- 不是银弹，不能完全防止逆向工程
- 不能阻止所有形式的代码攻击
- 不能替代服务端安全措施

**能做什么**：
- 增加逆向工程的成本和难度
- 保护敏感的业务逻辑
- 防止简单的代码窃取
- 基本的反调试保护

**建议**：
- 结合其他安全措施使用
- 定期更新混淆策略
- 不要过度依赖客户端安全
- 核心逻辑应放在服务端

## 更新历史

### v2.0 (当前版本)
- 新增多种加密算法支持（AES-GCM、RC4、ChaCha20、XOR）
- 增强反调试机制
- 添加自毁和内存保护功能
- 优化控制流平坦化
- 改进代码压缩算法

### v1.0 (初始版本)
- 基础变量混淆
- 字符串加密
- 代码压缩
- 控制流平坦化

## 技术支持

如遇到问题，请检查：
1. Go版本是否 >= 1.16
2. 依赖是否正确安装（运行 `go mod tidy`）
3. 代码是否符合JavaScript语法规范
4. 混淆配置是否合理

如需进一步帮助，请参考项目文档或提交Issue。

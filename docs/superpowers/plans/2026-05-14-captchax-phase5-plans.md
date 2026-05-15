# CaptchaX 第五阶段：高级功能开发计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task by task. Steps use checkbox (- [ ]) syntax for tracking.

**Goal:** 打造业界领先的 AI 驱动行为验证系统，新增验证码类型、增强 AI 风控、完善多语言支持、实现企业级功能。

**Architecture:** 基于现有 CaptchaX 架构，增加新验证码模块、AI 风控引擎、SDK 增强、企业级功能模块。

**Tech Stack:** Go 1.21+, Gin, PostgreSQL 16, Redis 7, Vanilla JS, 多语言 SDK

---

## 大任务16：新增验证码类型 - 旋转验证

### 子任务1：实现旋转图片生成算法
- **Files:**
  - Create: `/workspace/captchax/internal/captcha/rotate/rotate.go`
  - Create: `/workspace/captchax/internal/captcha/rotate/types.go`
- **Steps:**
  - [ ] Step 1: 定义 RotateCaptcha 数据结构
  ```go
  package rotate
  
  import (
      "image"
      "math/rand"
  )
  
  type RotateCaptcha struct {
      ID        string
      Image     image.Image
      Angle     int // 旋转角度 0-359
  }
  ```
  - [ ] Step 2: 实现图片生成与旋转函数
  ```go
  func GenerateRotateImage() (*RotateCaptcha, error) {
      // 实现：
      // 1. 生成基础图片或选择背景图
      // 2. 随机生成目标角度
      // 3. 应用旋转变换
      // 4. 返回验证码数据
  }
  ```
  - [ ] Step 3: 实现基本的图片处理工具函数
  - [ ] Step 4: 测试图片生成功能

### 子任务2：实现旋转角度验证逻辑
- **Files:**
  - Create: `/workspace/captchax/internal/captcha/rotate/verify.go`
- **Steps:**
  - [ ] Step 1: 编写验证函数，允许±5度误差
  ```go
  func VerifyRotateAngle(actualAngle int, targetAngle int) bool {
      diff := abs(actualAngle - targetAngle)
      return diff <= 5 || diff >= 355
  }
  
  func abs(x int) int {
      if x < 0 {
          return -x
      }
      return x
  }
  ```
  - [ ] Step 2: 实现验证流程逻辑
  - [ ] Step 3: 编写验证边界条件测试

### 子任务3：实现 Redis 缓存管理
- **Files:**
  - Create: `/workspace/captchax/internal/captcha/rotate/cache.go`
- **Steps:**
  - [ ] Step 1: 实现验证码数据缓存结构
  ```go
  type RotateCacheData struct {
      ID        string
      TargetAngle int
      CreatedAt time.Time
  }
  ```
  - [ ] Step 2: 实现 Set 和 Get 方法
  - [ ] Step 3: 实现过期自动清理逻辑

### 子任务4：编写单元测试
- **Files:**
  - Create: `/workspace/captchax/internal/captcha/rotate/rotate_test.go`
- **Steps:**
  - [ ] Step 1: 编写图片生成测试
  - [ ] Step 2: 编写验证逻辑测试
  - [ ] Step 3: 编写缓存功能测试
  - [ ] Step 4: 确保测试覆盖率 > 80%

### 子任务5：集成到 API 层
- **Files:**
  - Modify: `/workspace/captchax/internal/api/handlers.go`
  - Modify: `/workspace/captchax/internal/api/routes.go`
- **Steps:**
  - [ ] Step 1: 添加旋转验证码生成 API
  - [ ] Step 2: 添加旋转验证码验证 API
  - [ ] Step 3: 编写 API 集成测试
  - [ ] Step 4: 更新 API 文档

---

## 大任务17：新增验证码类型 - 文字识别验证

### 子任务1：实现扭曲文字生成
- **Files:**
  - Create: `/workspace/captchax/internal/captcha/text/text.go`
  - Create: `/workspace/captchax/internal/captcha/text/types.go`
- **Steps:**
  - [ ] Step 1: 定义文字验证码数据结构
  - [ ] Step 2: 实现随机字符串生成（数字+字母）
  - [ ] Step 3: 实现文字扭曲算法
  - [ ] Step 4: 实现颜色随机化

### 子任务2：实现干扰线和噪点生成
- **Files:**
  - Modify: `/workspace/captchax/internal/captcha/text/text.go`
- **Steps:**
  - [ ] Step 1: 实现干扰线绘制函数
  - [ ] Step 2: 实现噪点添加函数
  - [ ] Step 3: 实现背景渐变效果
  - [ ] Step 4: 测试干扰效果

### 子任务3：实现文字识别验证逻辑
- **Files:**
  - Create: `/workspace/captchax/internal/captcha/text/verify.go`
- **Steps:**
  - [ ] Step 1: 实现验证码字符串比较（忽略大小写）
  - [ ] Step 2: 实现验证流程逻辑
  - [ ] Step 3: 实现防暴力破解限制

### 子任务4-6：缓存、测试、API 集成（类似于旋转验证）

---

## 大任务18：新增验证码类型 - 图标选择验证

### 子任务1：设计图标库 (100+ 图标)
- **Files:**
  - Create: `/workspace/captchax/internal/captcha/icon/icons.go`
  - Create: `/workspace/captchax/web/static/icons/` (SVG/PNG)
- **Steps:**
  - [ ] Step 1: 准备 100+ 常用图标（动物、物品、符号等）
  - [ ] Step 2: 定义图标元数据结构
  - [ ] Step 3: 实现图标分类管理

### 子任务2：实现图标随机组合生成
- **Files:**
  - Create: `/workspace/captchax/internal/captcha/icon/icon.go`
- **Steps:**
  - [ ] Step 1: 实现随机选择 3-5 个目标图标
  - [ ] Step 2: 实现混入 6-10 个干扰图标
  - [ ] Step 3: 实现图标网格布局
  - [ ] Step 4: 实现图标随机排列

### 子任务3-6：验证逻辑、缓存、测试、API 集成

---

## 大任务23：国际化与多语言支持

### 子任务1：新增多语言文件
- **Files:**
  - Modify: `/workspace/captchax/web/i18n/`
- **Steps:**
  - [ ] Step 1: 创建 fr.json (法语)
  - [ ] Step 2: 创建 de.json (德语)
  - [ ] Step 3: 创建 es.json (西班牙语)
  - [ ] Step 4: 创建 ru.json (俄语)
  - [ ] Step 5: 创建 ja.json (日语)
  - [ ] Step 6: 创建 ko.json (韩语)
  - [ ] Step 7: 创建 ar.json (阿拉伯语)

### 子任务2：实现多语言切换机制
- **Files:**
  - Modify: `/workspace/captchax/web/static/captchax.js`
  - Modify: `/workspace/captchax/internal/middleware/i18n.go` (新建)
- **Steps:**
  - [ ] Step 1: 服务端实现 i18n 中间件
  - [ ] Step 2: 前端实现语言检测与切换
  - [ ] Step 3: 实现 Cookie/Header 语言偏好存储
  - [ ] Step 4: 更新管理后台支持语言配置

### 子任务3-4：时区自适应、i18n 测试

---

## 并行执行策略
**第一批任务（同时开始）：**
1. 大任务16：旋转验证
2. 大任务17：文字识别验证  
3. 大任务18：图标选择验证
4. 大任务23：国际化与多语言

---

## 验收标准
- [ ] 所有验证码类型正常工作
- [ ] 所有测试通过
- [ ] 代码覆盖率 > 80%
- [ ] API 文档完整更新
- [ ] 性能无显著下降

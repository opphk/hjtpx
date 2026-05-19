# API 错误码说明文档 v15.0

## 概述

本文档定义了HJTPX v15.0系统中所有API错误码及其含义，便于开发者理解和处理API调用过程中可能遇到的各种错误情况。

## 更新日志

- **v15.0** (2026-05-19): 完善错误码分类，增加验证码相关错误码和处理建议
- **v11.0** (2026-05-18): 完善错误码分类，增加处理建议和示例
- **v1.0** (2024-01): 初始版本

## 错误码分类总览

| 分类 | 错误码范围 | HTTP状态码 | 说明 |
|------|-----------|-----------|------|
| 客户端错误 | 1000-1999 | 400 | 请求参数错误 |
| 认证授权错误 | 2000-2999 | 401/403 | 认证和权限问题 |
| 资源错误 | 3000-3999 | 404/409 | 资源不存在或冲突 |
| 业务逻辑错误 | 4000-4999 | 400 | 业务规则验证失败 |
| 服务器错误 | 5000-5999 | 500/503 | 服务端问题 |
| 验证码错误 | 6000-6999 | 400 | 验证码专用错误 |

## 错误码详细说明

### 1. 客户端错误 (1000-1999)

#### 1.1 请求参数错误 (1000-1049)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 1001 | 400 | Invalid request format | 请求格式错误 | 检查Content-Type是否为application/json |
| 1002 | 400 | Invalid parameter | 参数值不符合要求 | 检查参数类型和范围 |
| 1003 | 400 | Validation failed | 数据校验失败 | 根据details字段查看具体校验失败原因 |
| 1004 | 400 | Missing required field | 缺少必填字段 | 检查必填参数列表 |
| 1005 | 400 | Invalid JSON format | JSON格式错误 | 使用JSON validator验证JSON格式 |
| 1006 | 400 | Invalid email format | 邮箱格式不正确 | 使用标准邮箱格式 |
| 1007 | 400 | Invalid phone format | 手机号格式不正确 | 使用国际通用手机号格式 |
| 1008 | 400 | Password too weak | 密码强度不足 | 密码需包含大小写字母、数字和特殊字符 |
| 1009 | 400 | Password mismatch | 两次密码不一致 | 确认新密码和确认密码相同 |
| 1010 | 400 | Invalid date format | 日期格式错误 | 使用ISO 8601格式：YYYY-MM-DD |
| 1011 | 400 | Date out of range | 日期超出范围 | 检查日期是否在有效范围内 |
| 1012 | 400 | Invalid enum value | 枚举值无效 | 使用文档中定义的有效枚举值 |
| 1013 | 400 | String too long | 字符串长度超限 | 缩短字符串长度 |
| 1014 | 400 | String too short | 字符串长度不足 | 增加字符串长度 |
| 1015 | 400 | Invalid URL format | URL格式错误 | 使用标准URL格式 |
| 1016 | 400 | Invalid IP address | IP地址格式错误 | 使用有效的IPv4或IPv6地址 |
| 1017 | 400 | Number out of range | 数值超出范围 | 检查数值是否在允许范围内 |
| 1018 | 400 | Division by zero | 除数不能为零 | 修改除数参数值 |
| 1019 | 400 | Invalid array format | 数组格式错误 | 确保数组元素类型一致 |

#### 1.2 文件上传错误 (1050-1099)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 1050 | 400 | File too large | 文件大小超过限制 | 减小文件大小 |
| 1051 | 400 | Invalid file type | 文件类型不支持 | 使用支持的文件类型 |
| 1052 | 400 | File corrupted | 文件已损坏 | 重新上传文件 |
| 1053 | 400 | Upload failed | 文件上传失败 | 检查网络连接或服务器状态 |
| 1054 | 400 | Storage quota exceeded | 存储配额已用完 | 清理存储空间或升级套餐 |
| 1055 | 400 | Invalid image format | 图片格式错误 | 使用支持的图片格式(jpg,png,gif,webp) |
| 1056 | 400 | Image too large | 图片尺寸过大 | 减小图片尺寸 |
| 1057 | 400 | Malicious file detected | 检测到恶意文件 | 检查文件安全性 |
| 1058 | 400 | File not found | 文件不存在 | 确认文件路径是否正确 |
| 1059 | 400 | Directory not writable | 目录不可写 | 检查目录权限 |

#### 1.3 限流错误 (1100-1199)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 1100 | 429 | Rate limit exceeded | 请求频率超限 | 降低请求频率，使用指数退避策略 |
| 1101 | 429 | Too many requests | 请求过于频繁 | 等待一段时间后重试 |
| 1102 | 429 | IP rate limit exceeded | IP请求频率超限 | 降低IP请求频率 |
| 1103 | 429 | User rate limit exceeded | 用户请求频率超限 | 降低用户请求频率 |
| 1104 | 429 | API rate limit exceeded | API请求频率超限 | 等待冷却时间后重试 |
| 1105 | 429 | Concurrent limit exceeded | 并发请求超限 | 减少并发请求数 |
| 1106 | 429 | Daily quota exceeded | 日配额已用完 | 等待次日配额重置或升级套餐 |
| 1107 | 429 | Monthly quota exceeded | 月配额已用完 | 等待下月配额重置或升级套餐 |
| 1108 | 429 | Request timeout | 请求超时 | 增加超时时间或简化请求 |

### 2. 认证与授权错误 (2000-2999)

#### 2.1 认证错误 (2000-2049)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 2001 | 401 | Unauthorized | 未授权访问 | 登录后获取Token |
| 2002 | 401 | Invalid token | Token无效 | 重新登录获取新Token |
| 2003 | 401 | Token expired | Token已过期 | 使用refresh_token刷新Token |
| 2004 | 401 | Token not found | Token不存在 | 重新登录获取Token |
| 2005 | 401 | Invalid credentials | 用户名或密码错误 | 检查用户名和密码 |
| 2006 | 401 | Invalid signature | 签名无效 | 检查签名算法和密钥 |
| 2007 | 401 | Signature expired | 签名已过期 | 重新生成签名 |
| 2008 | 401 | Invalid API key | API密钥无效 | 检查API密钥是否正确 |
| 2009 | 401 | API key expired | API密钥已过期 | 更新API密钥 |
| 2010 | 401 | Invalid app key | 应用密钥无效 | 检查应用密钥 |
| 2011 | 401 | Missing authentication | 缺少认证信息 | 添加Authorization头 |
| 2012 | 401 | Authentication failed | 认证失败 | 重新进行认证 |
| 2013 | 401 | MFA required | 需要多因素认证 | 完成MFA验证 |
| 2014 | 401 | MFA code invalid | MFA验证码无效 | 检查MFA代码是否正确 |
| 2015 | 401 | MFA code expired | MFA验证码已过期 | 获取新的MFA验证码 |
| 2016 | 401 | Account locked | 账户已被锁定 | 联系客服解锁账户 |
| 2017 | 401 | Account disabled | 账户已被禁用 | 联系客服启用账户 |
| 2018 | 401 | Account not activated | 账户未激活 | 通过邮箱激活账户 |
| 2019 | 401 | Email not verified | 邮箱未验证 | 验证邮箱地址 |
| 2020 | 401 | Too many login attempts | 登录尝试次数过多 | 等待一段时间后重试 |
| 2021 | 401 | Password expired | 密码已过期 | 修改密码 |
| 2022 | 401 | Password recently changed | 密码最近已修改 | 使用新密码登录 |
| 2023 | 401 | Device not trusted | 设备不受信任 | 将设备添加到信任列表 |
| 2024 | 401 | IP not allowed | IP地址不允许 | 检查IP访问策略 |
| 2025 | 401 | Captcha required | 需要验证码 | 完成验证码验证 |

#### 2.2 授权错误 (2050-2099)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 2050 | 403 | Forbidden | 禁止访问 | 检查权限配置 |
| 2051 | 403 | Insufficient permissions | 权限不足 | 申请更高权限 |
| 2052 | 403 | Operation not allowed | 操作不允许 | 检查操作权限 |
| 2053 | 403 | Resource access denied | 资源访问被拒绝 | 检查资源权限 |
| 2054 | 403 | IP blacklisted | IP已被加入黑名单 | 联系客服解决 |
| 2055 | 403 | User blacklisted | 用户已被加入黑名单 | 联系客服解决 |
| 2056 | 403 | Access denied by rule | 被风控规则拦截 | 检查风控规则配置 |
| 2057 | 403 | Method not allowed | 请求方法不允许 | 使用正确的HTTP方法 |
| 2058 | 403 | Content type not allowed | 内容类型不允许 | 使用支持的内容类型 |
| 2059 | 403 | Resource deleted | 资源已被删除 | 确认资源状态 |
| 2060 | 403 | Resource suspended | 资源已被暂停 | 联系客服恢复资源 |
| 2061 | 403 | Trial expired | 试用期已结束 | 升级到付费版本 |
| 2062 | 403 | Subscription expired | 订阅已过期 | 续订服务 |
| 2063 | 403 | Payment required | 需要支付 | 完成支付 |

### 3. 资源错误 (3000-3999)

#### 3.1 资源不存在 (3000-3049)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 3001 | 404 | Resource not found | 资源不存在 | 检查资源ID是否正确 |
| 3002 | 404 | User not found | 用户不存在 | 检查用户ID |
| 3003 | 404 | Application not found | 应用不存在 | 检查应用ID |
| 3004 | 404 | Session not found | 会话不存在 | 检查会话ID |
| 3005 | 404 | Captcha not found | 验证码不存在 | 检查验证码ID |
| 3006 | 404 | Rule not found | 规则不存在 | 检查规则ID |
| 3007 | 404 | Config not found | 配置不存在 | 检查配置ID |
| 3008 | 404 | Backup not found | 备份不存在 | 检查备份ID |
| 3009 | 404 | Log not found | 日志不存在 | 检查日志ID |
| 3010 | 404 | File not found | 文件不存在 | 检查文件路径 |
| 3011 | 404 | Endpoint not found | 接口不存在 | 检查接口路径 |
| 3012 | 404 | Template not found | 模板不存在 | 检查模板ID |
| 3013 | 404 | Image not found | 图片不存在 | 检查图片URL |
| 3014 | 404 | Video not found | 视频不存在 | 检查视频URL |
| 3015 | 404 | Audio not found | 音频不存在 | 检查音频URL |

#### 3.2 资源冲突 (3050-3099)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 3050 | 409 | Resource already exists | 资源已存在 | 使用已存在的资源或创建新资源 |
| 3051 | 409 | Email already registered | 邮箱已被注册 | 使用其他邮箱或找回密码 |
| 3052 | 409 | Username already taken | 用户名已被占用 | 使用其他用户名 |
| 3053 | 409 | Phone already registered | 手机号已被注册 | 使用其他手机号 |
| 3054 | 409 | App key already exists | 应用密钥已存在 | 使用已存在的应用 |
| 3055 | 409 | Domain already registered | 域名已被注册 | 使用其他域名 |
| 3056 | 409 | Duplicate entry | 重复条目 | 修改为唯一值 |
| 3057 | 409 | Resource locked | 资源已被锁定 | 等待解锁或联系管理员 |
| 3058 | 409 | Resource busy | 资源正忙 | 稍后重试 |
| 3059 | 409 | Version conflict | 版本冲突 | 使用最新版本 |
| 3060 | 409 | State conflict | 状态冲突 | 修改资源状态后再操作 |

#### 3.3 资源状态错误 (3100-3199)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 3100 | 400 | Resource inactive | 资源未激活 | 激活资源 |
| 3101 | 400 | Resource deleted | 资源已删除 | 恢复或重新创建 |
| 3102 | 400 | Resource expired | 资源已过期 | 续费或重新创建 |
| 3103 | 400 | Resource pending | 资源处理中 | 等待处理完成 |
| 3104 | 400 | Resource disabled | 资源已禁用 | 启用资源 |
| 3105 | 400 | Resource readonly | 资源只读 | 检查资源状态 |

### 4. 验证码错误 (6000-6999)

#### 4.1 验证码会话错误 (6000-6049)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 6001 | 400 | Captcha expired | 验证码已过期 | 重新获取验证码 |
| 6002 | 400 | Captcha invalid | 验证码不正确 | 检查验证码答案 |
| 6003 | 400 | Captcha already used | 验证码已使用 | 获取新验证码 |
| 6004 | 400 | Captcha not found | 验证码不存在 | 检查验证码ID |
| 6005 | 400 | Captcha already verified | 验证码已验证 | 不需要重复验证 |
| 6006 | 400 | Captcha attempts exceeded | 验证次数超限 | 稍后重试 |
| 6007 | 400 | Captcha time expired | 验证码时间已过期 | 加快验证速度或重新获取 |
| 6008 | 400 | Captcha token invalid | 验证码Token无效 | 重新获取验证码 |
| 6009 | 400 | Captcha answer required | 需要验证码答案 | 提供验证码答案 |
| 6010 | 400 | Captcha answer incorrect | 验证码答案错误 | 重试验证码 |
| 6011 | 400 | Captcha type not supported | 验证码类型不支持 | 使用支持的验证码类型 |
| 6012 | 400 | Captcha generation failed | 验证码生成失败 | 稍后重试 |
| 6013 | 400 | Captcha verification failed | 验证码验证失败 | 检查验证码状态 |
| 6014 | 400 | Captcha disabled | 验证码已禁用 | 启用验证码 |
| 6015 | 400 | Captcha session expired | 验证码会话已过期 | 重新获取验证码 |
| 6016 | 400 | Captcha session invalid | 验证码会话无效 | 重新获取验证码 |

#### 4.2 验证码类型错误 (6050-6099)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 6050 | 400 | Invalid slider position | 滑块位置无效 | 提供正确的滑块位置 |
| 6051 | 400 | Invalid click points | 点击位置无效 | 提供正确的点击位置 |
| 6052 | 400 | Invalid gesture pattern | 手势图案无效 | 提供正确的手势图案 |
| 6053 | 400 | Invalid rotation angle | 旋转角度无效 | 提供正确的旋转角度 |
| 6054 | 400 | Invalid voice answer | 语音答案无效 | 提供正确的语音验证码答案 |
| 6055 | 400 | Invalid lianliankan pairs | 连连看配对无效 | 提供正确的连连看配对 |
| 6056 | 400 | Too many wrong attempts | 错误次数过多 | 等待冷却后重试 |
| 6057 | 400 | Captcha already passed | 验证码已通过 | 不需要重复验证 |
| 6058 | 400 | Captcha blocked | 验证码被拦截 | 联系管理员解决 |
| 6059 | 400 | Captcha requires mouse | 验证码需要鼠标操作 | 使用鼠标完成验证 |

### 5. 业务逻辑错误 (4000-4999)

#### 5.1 业务规则错误 (4000-4049)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 4001 | 400 | Risk score too high | 风险评分过高 | 改善操作行为 |
| 4002 | 400 | Verification failed | 验证失败 | 重试验证 |
| 4003 | 400 | Biometric mismatch | 生物特征不匹配 | 重新采集生物特征 |
| 4004 | 400 | Not enough samples | 样本数量不足 | 提供更多样本数据 |
| 4005 | 400 | Profile not found | 用户档案不存在 | 创建用户档案 |
| 4006 | 400 | Profile not registered | 用户档案未注册 | 完成注册流程 |
| 4007 | 400 | Operation not allowed | 操作不允许 | 检查操作权限 |
| 4008 | 400 | Maintenance mode | 系统维护中 | 等待维护结束 |
| 4009 | 400 | Feature disabled | 功能已禁用 | 联系管理员启用 |
| 4010 | 400 | Quota exceeded | 配额已用尽 | 升级套餐或等待配额重置 |
| 4011 | 400 | Subscription required | 需要订阅 | 订阅服务 |
| 4012 | 400 | Trial ended | 试用期已结束 | 升级到付费版本 |
| 4013 | 400 | Payment failed | 支付失败 | 检查支付方式 |
| 4014 | 400 | Refund not allowed | 不允许退款 | 检查退款政策 |
| 4015 | 400 | Withdrawal not allowed | 不允许提现 | 检查提现条件 |
| 4016 | 400 | Transfer not allowed | 不允许转账 | 检查转账条件 |
| 4017 | 400 | Purchase not allowed | 不允许购买 | 检查购买条件 |
| 4018 | 400 | Age verification required | 需要年龄验证 | 完成年龄验证 |
| 4019 | 400 | Identity verification required | 需要身份验证 | 完成身份验证 |

#### 5.2 数据处理错误 (4050-4099)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 4050 | 400 | Data processing error | 数据处理错误 | 检查数据格式 |
| 4051 | 400 | Data validation failed | 数据校验失败 | 检查数据有效性 |
| 4052 | 400 | Data format error | 数据格式错误 | 使用正确的数据格式 |
| 4053 | 400 | Data conversion failed | 数据转换失败 | 检查数据类型 |
| 4054 | 400 | Data serialization failed | 数据序列化失败 | 检查数据结构 |
| 4055 | 400 | Data deserialization failed | 数据反序列化失败 | 检查数据格式 |
| 4056 | 400 | Data corruption detected | 检测到数据损坏 | 重新提供数据 |
| 4057 | 400 | Data integrity check failed | 数据完整性检查失败 | 重新提交数据 |
| 4058 | 400 | Data size limit exceeded | 数据大小超限 | 减小数据大小 |
| 4059 | 400 | Data encoding error | 数据编码错误 | 使用正确的编码格式 |

### 6. 服务器错误 (5000-5999)

#### 6.1 系统错误 (5000-5049)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 5001 | 500 | Internal server error | 服务器内部错误 | 联系技术支持 |
| 5002 | 500 | Database error | 数据库操作失败 | 检查数据库状态，稍后重试 |
| 5003 | 500 | Cache error | 缓存操作失败 | 检查缓存服务，稍后重试 |
| 5004 | 500 | External API error | 外部API调用失败 | 稍后重试 |
| 5005 | 503 | Service unavailable | 服务暂时不可用 | 稍后重试 |
| 5006 | 500 | Timeout | 请求超时 | 增加超时时间或简化请求 |
| 5007 | 500 | Encryption error | 加密操作失败 | 稍后重试 |
| 5008 | 500 | Decryption error | 解密操作失败 | 检查加密数据 |
| 5009 | 500 | Backup failed | 备份操作失败 | 检查存储空间，稍后重试 |
| 5010 | 500 | Restore failed | 恢复操作失败 | 检查备份文件完整性 |
| 5011 | 500 | Export failed | 导出操作失败 | 检查导出参数，稍后重试 |
| 5012 | 500 | Import failed | 导入操作失败 | 检查导入文件格式 |
| 5013 | 500 | Processing failed | 处理失败 | 简化请求，稍后重试 |
| 5014 | 500 | Computation error | 计算错误 | 检查输入参数 |
| 5015 | 500 | Validation error | 验证错误 | 检查输入数据 |
| 5016 | 500 | Transformation error | 转换错误 | 检查数据格式 |
| 5017 | 500 | Serialization error | 序列化错误 | 检查数据结构 |
| 5018 | 500 | Parsing error | 解析错误 | 检查数据格式 |
| 5019 | 500 | Compilation error | 编译错误 | 联系技术支持 |

#### 6.2 资源错误 (5050-5099)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 5050 | 500 | Out of memory | 内存不足 | 优化内存使用 |
| 5051 | 500 | Out of disk space | 磁盘空间不足 | 清理磁盘空间 |
| 5052 | 500 | CPU overload | CPU过载 | 降低请求频率 |
| 5053 | 500 | Network error | 网络错误 | 检查网络连接 |
| 5054 | 500 | File system error | 文件系统错误 | 检查文件系统 |
| 5055 | 500 | Process limit exceeded | 进程数超限 | 减少并发请求 |
| 5056 | 500 | Connection limit exceeded | 连接数超限 | 减少连接数 |
| 5057 | 500 | Thread limit exceeded | 线程数超限 | 优化并发处理 |
| 5058 | 500 | Lock timeout | 锁超时 | 稍后重试 |
| 5059 | 500 | Deadlock detected | 检测到死锁 | 稍后重试 |

#### 6.3 配置错误 (5100-5149)

| 错误码 | HTTP状态码 | 错误信息 | 说明 | 处理建议 |
| :--- | :--- | :--- | :--- | :--- |
| 5100 | 500 | Configuration error | 配置错误 | 检查配置文件 |
| 5101 | 500 | Config not found | 配置不存在 | 检查配置路径 |
| 5102 | 500 | Config invalid | 配置无效 | 检查配置内容 |
| 5103 | 500 | Config corrupted | 配置已损坏 | 恢复默认配置 |
| 5104 | 500 | Config locked | 配置已锁定 | 管理员解锁后修改 |
| 5105 | 500 | Config outdated | 配置已过期 | 刷新配置 |
| 5106 | 500 | Environment error | 环境错误 | 检查环境配置 |
| 5107 | 500 | Dependency error | 依赖错误 | 检查依赖服务 |
| 5108 | 500 | License error | 许可证错误 | 检查许可证有效性 |
| 5109 | 500 | Permission denied | 权限被拒绝 | 检查文件权限 |

## 错误响应格式

所有API错误响应遵循统一格式：

```json
{
  "code": 1001,
  "message": "Invalid request format",
  "details": "Content-Type must be application/json",
  "timestamp": "2026-05-19T10:30:00Z",
  "request_id": "req_abc123def456",
  "trace_id": "trace_xyz789",
  "docs_url": "https://docs.hjtpx.com/errors/1001"
}
```

### 字段说明

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| code | integer | 错误码 |
| message | string | 错误简要描述 |
| details | string | 错误详细信息（可选） |
| timestamp | string | 错误发生时间（ISO 8601格式） |
| request_id | string | 请求唯一标识（用于日志追踪） |
| trace_id | string | 追踪ID（用于分布式追踪） |
| docs_url | string | 错误文档链接（可选） |

## 错误处理建议

### 1. 客户端错误(10xx)处理

客户端错误通常是由于请求参数或格式问题导致的。请按以下步骤处理：

1. **检查错误详情**：查看`details`字段了解具体问题
2. **验证请求格式**：确保Content-Type为`application/json`
3. **检查参数完整性**：确保所有必填参数都已提供
4. **验证参数格式**：确保参数类型和范围正确
5. **使用示例代码**：参考API文档中的示例

### 2. 认证错误(20xx)处理

认证错误通常需要重新获取认证凭证。请按以下步骤处理：

1. **检查Token有效性**：验证Token是否过期或被撤销
2. **使用Refresh Token**：如果Token过期，使用refresh_token刷新
3. **重新登录**：如果Refresh Token也过期，需要重新登录
4. **检查权限**：确保账户有足够的权限访问资源
5. **完成MFA验证**：如果需要MFA验证，完成验证流程

### 3. 资源错误(30xx)处理

资源错误通常是资源不存在或冲突。请按以下步骤处理：

1. **检查资源ID**：确保资源ID正确
2. **验证资源状态**：检查资源是否被删除、禁用或过期
3. **处理冲突**：如果是重复资源，使用已存在的资源或创建新资源
4. **恢复资源**：如果资源被误删，尝试恢复

### 4. 验证码错误(60xx)处理

验证码错误需要重新获取或验证验证码。请按以下步骤处理：

1. **重新获取验证码**：如果是过期或已使用，重新获取
2. **检查验证码答案**：确保验证码答案正确
3. **提供轨迹数据**：确保提供完整的轨迹数据
4. **等待冷却时间**：如果验证次数超限，等待后重试
5. **使用备用方案**：如果验证码持续失败，使用其他验证方式

### 5. 业务逻辑错误(40xx)处理

业务逻辑错误需要根据具体业务规则处理。请按以下步骤处理：

1. **了解业务规则**：阅读相关业务规则文档
2. **检查业务状态**：确保满足业务前置条件
3. **改善风险评分**：如果是风险评分过高，改善操作行为
4. **完成必要验证**：如果需要身份验证或年龄验证，完成验证
5. **升级套餐**：如果是配额或订阅问题，升级套餐

### 6. 服务器错误(50xx)处理

服务器错误通常是临时性问题。请按以下步骤处理：

1. **稍后重试**：大多数服务器错误是临时的，稍后重试可能成功
2. **使用指数退避**：逐步增加重试间隔时间
3. **检查服务状态**：查看系统状态页面了解服务健康状况
4. **联系技术支持**：如果问题持续存在，联系技术支持
5. **记录错误信息**：保存错误信息以便排查问题

## 示例

### 示例1：参数验证失败

```json
{
  "code": 1003,
  "message": "Validation failed",
  "details": "Field 'email' must be a valid email address",
  "timestamp": "2026-05-19T10:30:00Z",
  "request_id": "req_12345",
  "docs_url": "https://docs.hjtpx.com/errors/1003"
}
```

### 示例2：Token过期

```json
{
  "code": 2003,
  "message": "Token expired",
  "details": "Your access token has expired. Please refresh it using the refresh_token.",
  "timestamp": "2026-05-19T10:30:00Z",
  "request_id": "req_67890",
  "docs_url": "https://docs.hjtpx.com/errors/2003"
}
```

### 示例3：资源不存在

```json
{
  "code": 3001,
  "message": "Resource not found",
  "details": "Application with id 999 does not exist",
  "timestamp": "2026-05-19T10:30:00Z",
  "request_id": "req_abcde",
  "docs_url": "https://docs.hjtpx.com/errors/3001"
}
```

### 示例4：验证码过期

```json
{
  "code": 6001,
  "message": "Captcha expired",
  "details": "The captcha session has expired. Please generate a new captcha.",
  "timestamp": "2026-05-19T10:30:00Z",
  "request_id": "req_fghij",
  "docs_url": "https://docs.hjtpx.com/errors/6001"
}
```

### 示例5：限流错误

```json
{
  "code": 1100,
  "message": "Rate limit exceeded",
  "details": "You have exceeded the rate limit of 1000 requests per minute. Please wait 60 seconds.",
  "timestamp": "2026-05-19T10:30:00Z",
  "request_id": "req_klmno",
  "retry_after": 60,
  "docs_url": "https://docs.hjtpx.com/errors/1100"
}
```

### 示例6：服务器错误

```json
{
  "code": 5001,
  "message": "Internal server error",
  "details": "An unexpected error occurred. Our team has been notified. Please try again later.",
  "timestamp": "2026-05-19T10:30:00Z",
  "request_id": "req_pqrst",
  "docs_url": "https://docs.hjtpx.com/errors/5001"
}
```

---

## 错误处理最佳实践

### 1. 统一错误处理

```go
// Go示例：统一错误处理
type ErrorResponse struct {
    Code      int    `json:"code"`
    Message   string `json:"message"`
    Details   string `json:"details,omitempty"`
    Timestamp string `json:"timestamp"`
    RequestID string `json:"request_id"`
    DocsURL   string `json:"docs_url,omitempty"`
}

func handleError(w http.ResponseWriter, err error) {
    var code int
    var message string
    
    switch {
    case errors.Is(err, ErrInvalidRequest):
        code = 1001
        message = "Invalid request"
    case errors.Is(err, ErrUnauthorized):
        code = 2001
        message = "Unauthorized"
    case errors.Is(err, ErrResourceNotFound):
        code = 3001
        message = "Resource not found"
    case errors.Is(err, ErrRateLimitExceeded):
        code = 1100
        message = "Rate limit exceeded"
    default:
        code = 5001
        message = "Internal server error"
    }
    
    response := ErrorResponse{
        Code:      code,
        Message:   message,
        Details:   err.Error(),
        Timestamp: time.Now().Format(time.RFC3339),
        RequestID: getRequestID(),
        DocsURL:   fmt.Sprintf("https://docs.hjtpx.com/errors/%d", code),
    }
    
    w.WriteHeader(getHTTPStatus(code))
    json.NewEncoder(w).Encode(response)
}

func getHTTPStatus(code int) int {
    switch {
    case code >= 1000 && code < 2000:
        return http.StatusBadRequest
    case code >= 2000 && code < 3000:
        return http.StatusUnauthorized
    case code >= 3000 && code < 4000:
        return http.StatusConflict
    case code >= 4000 && code < 5000:
        return http.StatusBadRequest
    case code >= 5000:
        return http.StatusInternalServerError
    case code >= 6000:
        return http.StatusBadRequest
    default:
        return http.StatusInternalServerError
    }
}
```

### 2. 重试策略

```go
// Go示例：指数退避重试
func withRetry(ctx context.Context, fn func() error, maxRetries int) error {
    var err error
    
    for i := 0; i < maxRetries; i++ {
        err = fn()
        if err == nil {
            return nil
        }
        
        // 判断是否可重试
        if !isRetryable(err) {
            return err
        }
        
        // 指数退避
        backoff := time.Duration(math.Pow(2, float64(i))) * 100 * time.Millisecond
        maxBackoff := 30 * time.Second
        if backoff > maxBackoff {
            backoff = maxBackoff
        }
        
        select {
        case <-time.After(backoff):
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    return fmt.Errorf("重试%d次后仍然失败: %w", maxRetries, err)
}

func isRetryable(err error) bool {
    // 网络错误、超时、服务器错误可重试
    if os.IsTimeout(err) || os.IsTemporary(err) {
        return true
    }
    
    // 5xx服务器错误可重试
    if err, ok := err.(interface{ Code() int }); ok {
        code := err.Code()
        if code >= 5000 && code < 6000 {
            return true
        }
    }
    
    // 限流错误可重试
    if err, ok := err.(interface{ Code() int }); ok {
        code := err.Code()
        if code >= 1100 && code < 1200 {
            return true
        }
    }
    
    return false
}
```

### 3. 前端错误处理

```javascript
// JavaScript示例：统一错误处理
class ApiError extends Error {
    constructor(code, message, details = {}) {
        super(message);
        this.code = code;
        this.details = details;
        this.timestamp = new Date().toISOString();
        this.docsUrl = `https://docs.hjtpx.com/errors/${code}`;
    }
}

async function handleApiRequest(request) {
    try {
        const response = await fetch(request);
        const data = await response.json();
        
        if (data.code !== 0) {
            const error = new ApiError(data.code, data.message, data);
            
            // 根据错误码处理
            switch (data.code) {
                case 1001: // 参数错误
                case 1002:
                case 1003:
                    showError(`参数错误: ${data.message}`);
                    highlightInvalidFields(data.details);
                    break;
                    
                case 1100: // 限流错误
                    const retryAfter = data.retry_after || 60;
                    showWarning(`请求过于频繁，请在${retryAfter}秒后重试`);
                    setTimeout(() => retryRequest(request), retryAfter * 1000);
                    break;
                    
                case 2003: // Token过期
                    showInfo('登录已过期，请重新登录');
                    redirectToLogin();
                    break;
                    
                case 3001: // 资源不存在
                    showError('请求的资源不存在');
                    break;
                    
                case 6001: // 验证码过期
                    showError('验证码已过期，请重新获取');
                    refreshCaptcha();
                    break;
                    
                case 5001: // 服务器错误
                    showError('服务器错误，请稍后重试');
                    break;
                    
                default:
                    showError(`请求失败: ${data.message}`);
            }
            
            // 记录错误
            logger.error('API Error', {
                code: data.code,
                message: data.message,
                request_id: data.request_id,
                url: data.docs_url
            });
            
            return null;
        }
        
        return data.data;
    } catch (error) {
        if (error instanceof ApiError) {
            throw error;
        }
        
        console.error('网络错误:', error);
        showError('网络错误，请检查网络连接');
        logger.error('Network Error', error);
        return null;
    }
}
```

### 4. 错误日志记录

```go
// Go示例：结构化日志记录
func logError(requestID string, err error, context map[string]interface{}) {
    log.WithFields(log.Fields{
        "request_id": requestID,
        "error_code": getErrorCode(err),
        "error_message": err.Error(),
        "error_type": reflect.TypeOf(err).String(),
        "context": context,
        "timestamp": time.Now(),
        "service": "hjtpx-api",
        "version": "15.0.0",
    }).Error("API Error")
    
    // 发送到错误追踪系统
    sendToErrorTracker(err, requestID)
}

func getErrorCode(err error) int {
    if apiErr, ok := err.(interface{ Code() int }); ok {
        return apiErr.Code()
    }
    return 5001
}
```

### 5. 错误监控告警

```yaml
# Prometheus告警规则
groups:
  - name: hjtpx-errors
    rules:
      - alert: HighClientErrorRate
        expr: |
          rate(http_requests_total{status=~"4.."}[5m]) /
          rate(http_requests_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "客户端错误率过高"
          description: "客户端错误率超过10%"
          runbook_url: "https://docs.hjtpx.com/runbooks/high-client-error-rate"
      
      - alert: HighServerErrorRate
        expr: |
          rate(http_requests_total{status=~"5.."}[5m]) /
          rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "服务器错误率过高"
          description: "服务器错误率超过5%"
          runbook_url: "https://docs.hjtpx.com/runbooks/high-server-error-rate"
      
      - alert: HighCaptchaErrorRate
        expr: |
          rate(http_requests_total{endpoint="/captcha/*",status=~"4.."}[5m]) /
          rate(http_requests_total{endpoint="/captcha/*"}[5m]) > 0.2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "验证码错误率过高"
          description: "验证码错误率超过20%"
          runbook_url: "https://docs.hjtpx.com/runbooks/high-captcha-error-rate"
```

---

## 常见问题排查

### Q1: 验证码验证失败率高怎么办？

**原因分析：**

1. 客户端时间不准确（时间偏差可能导致签名验证失败）
2. 网络延迟过大（考虑增加超时时间）
3. 轨迹数据格式不正确（确保轨迹数据符合规范）
4. 服务端处理异常（查看服务端日志）

**排查步骤：**

```bash
# 1. 检查客户端时间
date
timedatectl

# 2. 测试网络延迟
ping api.hjtpx.com
traceroute api.hjtpx.com

# 3. 查看服务端日志
docker-compose logs backend | grep -i captcha
docker-compose logs backend | grep -i error

# 4. 测试验证码接口
curl -X POST http://localhost:8080/api/v1/captcha/slider \
  -H "Content-Type: application/json"
```

**解决方案：**

- 确保客户端时间与服务器时间同步（误差在30秒内）
- 增加网络超时时间
- 按照文档规范生成轨迹数据
- 如果问题持续，联系技术支持

### Q2: Token过期频繁？

**原因分析：**

1. Token有效期设置过短
2. 客户端没有实现Token自动刷新
3. 客户端缓存了过期Token

**解决方案：**

```javascript
// 正确实现Token自动刷新
class TokenManager {
    constructor() {
        this.accessToken = null;
        this.refreshToken = null;
        this.tokenExpiresAt = null;
    }
    
    async getAccessToken() {
        if (this.isTokenExpired()) {
            await this.refreshAccessToken();
        }
        return this.accessToken;
    }
    
    isTokenExpired() {
        if (!this.tokenExpiresAt) return true;
        // 提前5分钟刷新
        return Date.now() >= this.tokenExpiresAt - 5 * 60 * 1000;
    }
    
    async refreshAccessToken() {
        const response = await fetch('/api/v1/auth/refresh', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: this.refreshToken })
        });
        
        if (!response.ok) {
            // Refresh Token也过期，需要重新登录
            this.redirectToLogin();
            return;
        }
        
        const data = await response.json();
        this.accessToken = data.access_token;
        this.refreshToken = data.refresh_token;
        this.tokenExpiresAt = Date.now() + data.expires_in * 1000;
    }
}
```

### Q3: 限流错误频繁？

**原因分析：**

1. 请求频率确实过高
2. 存在异常请求或爬虫
3. 限流阈值设置不当

**解决方案：**

```javascript
// 正确实现限流控制
class RateLimiter {
    constructor(maxRequests, windowMs) {
        this.maxRequests = maxRequests;
        this.windowMs = windowMs;
        this.requests = [];
    }
    
    canMakeRequest() {
        const now = Date.now();
        this.requests = this.requests.filter(t => now - t < this.windowMs);
        return this.requests.length < this.maxRequests;
    }
    
    async makeRequest(fn) {
        if (!this.canMakeRequest()) {
            const waitTime = this.windowMs - (Date.now() - this.requests[0]);
            await new Promise(resolve => setTimeout(resolve, waitTime));
        }
        
        this.requests.push(Date.now());
        return fn();
    }
    
    // 使用指数退避
    async withRetry(fn, maxRetries = 3) {
        for (let i = 0; i < maxRetries; i++) {
            try {
                return await this.makeRequest(fn);
            } catch (error) {
                if (error.code === 1100) { // 限流错误
                    const backoff = Math.pow(2, i) * 1000;
                    await new Promise(resolve => setTimeout(resolve, backoff));
                    continue;
                }
                throw error;
            }
        }
        throw new Error('Max retries exceeded');
    }
}
```

### Q4: 如何获取错误码文档？

**解决方案：**

所有错误响应都包含`docs_url`字段，指向对应的错误文档：

```json
{
  "code": 1001,
  "message": "Invalid request format",
  "docs_url": "https://docs.hjtpx.com/errors/1001"
}
```

访问该URL可以获取：

- 错误码的详细说明
- 错误原因分析
- 处理建议和步骤
- 相关代码示例
- 常见问题解答

---

## 相关文档

- [API接口文档](./API接口文档.md)
- [OpenAPI规范](./openapi.yaml)
- [部署文档](./部署文档.md)
- [开发者指南](./开发者指南.md)
- [故障排查手册](./故障排查手册.md)
- [运维手册](./运维手册.md)

---

## 错误码维护

### 如何报告新错误码？

如果您在开发过程中遇到未文档化的错误码，请按以下步骤处理：

1. **记录错误信息**：保存完整的错误响应
2. **联系技术支持**：提供错误码和错误信息
3. **提交Issue**：在GitHub仓库提交Issue

### 如何申请新错误码？

如果现有错误码无法满足业务需求，可以申请添加新错误码：

1. **准备提案**：说明错误码的用途和分类
2. **提交PR**：在GitHub仓库提交PR
3. **审核流程**：经过技术评审后合并

---

*文档版本: 15.0*  
*最后更新: 2026-05-19*

## 错误响应格式

所有API错误响应遵循统一格式：

```json
{
  "code": 1001,
  "message": "Invalid request",
  "details": "The request body contains invalid JSON",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "abc123def456"
}
```

### 字段说明

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| code | integer | 错误码 |
| message | string | 错误简要描述 |
| details | string | 错误详细信息（可选） |
| timestamp | string | 错误发生时间（ISO 8601格式） |
| request_id | string | 请求唯一标识（用于日志追踪） |

## 错误处理建议

1. **客户端错误(10xx)**: 检查请求参数是否正确，确保所有必填字段都已提供且格式正确。

2. **认证错误(20xx)**: 检查Token是否有效，必要时重新获取Token或重新登录。

3. **资源错误(30xx)**: 确认资源ID是否正确，检查资源是否存在或是否已被删除。

4. **服务器错误(50xx)**: 记录错误信息并稍后重试，如问题持续存在请联系系统管理员。

5. **业务错误(40xx)**: 根据具体错误信息调整业务逻辑或用户操作。

## 示例

### 示例1：参数验证失败

```json
{
  "code": 1003,
  "message": "Validation failed",
  "details": "Email format is invalid",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req-12345"
}
```

### 示例2：Token过期

```json
{
  "code": 2003,
  "message": "Token expired",
  "details": "Please refresh your token",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req-67890"
}
```

### 示例3：资源不存在

```json
{
  "code": 3001,
  "message": "Resource not found",
  "details": "Application with id 999 does not exist",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req-abcde"
}
```

---

## 错误处理最佳实践

### 1. 客户端错误处理

```go
// Go示例
func handleClientError(code int, message string) error {
    switch code {
    case 1001:
        return fmt.Errorf("请求参数格式错误: %s", message)
    case 1002:
        return fmt.Errorf("参数值不符合要求: %s", message)
    case 1003:
        return fmt.Errorf("数据校验失败: %s", message)
    case 1009:
        return fmt.Errorf("验证码已过期，请重新获取")
    case 1010:
        return fmt.Errorf("验证码不正确，请重试")
    default:
        return fmt.Errorf("客户端错误 [%d]: %s", code, message)
    }
}

// Java示例
public class ErrorHandler {
    public static String handleClientError(int code, String message) {
        switch (code) {
            case 1001:
                return "请求参数格式错误: " + message;
            case 1002:
                return "参数值不符合要求: " + message;
            case 1003:
                return "数据校验失败: " + message;
            case 1009:
                return "验证码已过期，请重新获取";
            case 1010:
                return "验证码不正确，请重试";
            default:
                return String.format("客户端错误 [%d]: %s", code, message);
        }
    }
}

// Python示例
def handle_client_error(code: int, message: str) -> str:
    error_messages = {
        1001: "请求参数格式错误",
        1002: "参数值不符合要求",
        1003: "数据校验失败",
        1009: "验证码已过期，请重新获取",
        1010: "验证码不正确，请重试",
    }
    return error_messages.get(code, f"客户端错误 [{code}]: {message}")
```

### 2. 认证错误处理

```go
// 重定向到登录页面
func handleAuthError(w http.ResponseWriter, code int) {
    switch code {
    case 2001, 2002, 2003, 2004:
        // Token无效或过期，重定向到登录
        http.Redirect(w, r, "/login?redirect="+url.QueryEscape(r.RequestURI), 302)
    case 2006:
        // 权限不足
        http.Error(w, "权限不足", http.StatusForbidden)
    case 2008, 2009:
        // 账户被锁定或禁用
        http.Error(w, "账户状态异常，请联系客服", http.StatusForbidden)
    }
}
```

### 3. 重试策略

```go
// 指数退避重试
func withRetry(fn func() error, maxRetries int) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        err = fn()
        if err == nil {
            return nil
        }

        // 判断是否可重试
        if !isRetryable(err) {
            return err
        }

        // 指数退避
        backoff := time.Duration(math.Pow(2, float64(i))) * 100 * time.Millisecond
        time.Sleep(backoff)
    }

    return fmt.Errorf("重试%d次后仍然失败: %v", maxRetries, err)
}

func isRetryable(err error) bool {
    // 网络错误、超时、服务器错误可重试
    if os.IsTimeout(err) || os.IsTemporary(err) {
        return true
    }

    // 500系列错误可重试
    if strings.Contains(err.Error(), "500") {
        return true
    }

    // 429限流错误可重试
    if strings.Contains(err.Error(), "429") {
        return true
    }

    return false
}
```

### 4. 错误日志记录

```go
// 结构化日志记录
func logError(requestID string, code int, message string, err error) {
    log.WithFields(log.Fields{
        "request_id": requestID,
        "error_code": code,
        "message":    message,
        "error":      err.Error(),
        "timestamp":  time.Now(),
    }).Error("API Error")
}
```

### 5. 前端错误处理

```javascript
// 统一错误处理
async function handleApiRequest(request) {
    try {
        const response = await fetch(request);
        const data = await response.json();

        if (data.code !== 0) {
            switch (data.code) {
                case 1001:
                case 1002:
                case 1003:
                    // 参数错误，提示用户修正
                    showError(`参数错误: ${data.message}`);
                    break;
                case 1009:
                case 1010:
                    // 验证码错误，重新获取
                    refreshCaptcha();
                    showError('验证码已过期，请重新验证');
                    break;
                case 2002:
                case 2003:
                    // Token过期，重新登录
                    redirectToLogin();
                    break;
                case 40001:
                    // 限流，等待后重试
                    const retryAfter = data.data?.retry_after || 60;
                    showError(`请求过于频繁，请在${retryAfter}秒后重试`);
                    setTimeout(() => retryRequest(request), retryAfter * 1000);
                    break;
                default:
                    showError(`请求失败: ${data.message}`);
            }
            return null;
        }

        return data.data;
    } catch (error) {
        console.error('网络错误:', error);
        showError('网络错误，请检查网络连接');
        return null;
    }
}
```

---

## 常见问题排查

### Q1: 验证码验证失败率高怎么办？

1. 检查客户端时间是否准确（时间偏差可能导致签名验证失败）
2. 检查网络延迟是否过大（考虑增加超时时间）
3. 确认轨迹数据格式是否正确
4. 查看服务端日志中的详细错误信息

### Q2: Token过期频繁？

1. 检查Token有效期设置是否合理
2. 考虑实现Token自动刷新机制
3. 确认客户端没有缓存过期Token

### Q3: 限流错误频繁？

1. 检查是否有异常请求或爬虫
2. 实施客户端限流，避免触发服务端限制
3. 考虑升级服务套餐或联系技术支持

---

## 相关文档

- [API接口文档](../docs/API接口文档.md)
- [部署文档](../docs/部署文档.md)
- [开发者指南](../docs/开发者指南.md)

---

*文档版本: 11.0*  
*最后更新: 2026-05-18*

const swaggerJsdoc = require('swagger-jsdoc');

/**
 * @swagger
 * /api/v1/health:
 *   get:
 *     summary: 健康检查
 *     description: 验证服务是否正常运行，返回服务状态信息
 *     tags: [Health]
 *     security: []
 *     responses:
 *       200:
 *         description: 服务运行正常
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: true
 *                 data:
 *                   type: object
 *                   properties:
 *                     status:
 *                       type: string
 *                       example: healthy
 *                     service:
 *                       type: string
 *                       example: HJTPX API
 *                     version:
 *                       type: string
 *                       example: 1.0.0
 *                     timestamp:
 *                       type: string
 *                       format: date-time
 *                     uptime:
 *                       type: number
 *                       description: 服务运行时间（秒）
 *                     environment:
 *                       type: string
 *                       example: development
 *       503:
 *         description: 服务不可用
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */

/**
 * @swagger
 * /api/v1/health/detailed:
 *   get:
 *     summary: 详细健康检查
 *     description: 返回所有依赖服务（数据库、Redis、缓存）的详细状态
 *     tags: [Health]
 *     security: []
 *     responses:
 *       200:
 *         description: 服务运行正常或降级运行
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *                   properties:
 *                     status:
 *                       type: string
 *                       enum: [healthy, degraded, unhealthy]
 *                     timestamp:
 *                       type: string
 *                       format: date-time
 *                     service:
 *                       type: string
 *                     version:
 *                       type: string
 *                     uptime:
 *                       type: number
 *                     environment:
 *                       type: string
 *                     checks:
 *                       type: object
 *                       properties:
 *                         database:
 *                           $ref: '#/components/schemas/HealthCheck'
 *                         redis:
 *                           $ref: '#/components/schemas/HealthCheck'
 *                         cache:
 *                           $ref: '#/components/schemas/HealthCheck'
 *                         memory:
 *                           $ref: '#/components/schemas/HealthCheck'
 *                         cpu:
 *                           $ref: '#/components/schemas/HealthCheck'
 *       503:
 *         description: 服务不可用
 */

/**
 * @swagger
 * /api/v1/auth/login:
 *   post:
 *     summary: 用户登录
 *     description: 使用邮箱和密码登录系统，返回 JWT 访问令牌
 *     tags: [Authentication]
 *     security: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - email
 *               - password
 *             properties:
 *               email:
 *                 type: string
 *                 format: email
 *                 description: 用户邮箱地址
 *                 example: user@example.com
 *               password:
 *                 type: string
 *                 format: password
 *                 minLength: 8
 *                 maxLength: 128
 *                 description: 用户密码（至少8个字符，包含大小写字母和数字）
 *                 example: SecurePass123
 *     responses:
 *       200:
 *         description: 登录成功
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: true
 *                 data:
 *                   type: object
 *                   properties:
 *                     user:
 *                       $ref: '#/components/schemas/User'
 *                     token:
 *                       type: string
 *                       description: JWT 访问令牌
 *                       example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
 *                     expiresIn:
 *                       type: string
 *                       description: Token 过期时间
 *                       example: 7d
 *                 message:
 *                   type: string
 *                   example: Login successful
 *                 timestamp:
 *                   type: string
 *                   format: date-time
 *       401:
 *         description: 认证失败（无效的邮箱或密码）
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 *             example:
 *               success: false
 *               error:
 *                 code: UNAUTHORIZED
 *                 message: Invalid email or password
 *       429:
 *         description: 请求过于频繁（登录尝试次数限制）
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */

/**
 * @swagger
 * /api/v1/auth/register:
 *   post:
 *     summary: 用户注册
 *     description: 创建新用户账号，自动登录并返回 JWT 令牌
 *     tags: [Authentication]
 *     security: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - email
 *               - name
 *               - password
 *             properties:
 *               email:
 *                 type: string
 *                 format: email
 *                 description: 用户邮箱（必须是有效邮箱格式）
 *                 example: user@example.com
 *               name:
 *                 type: string
 *                 minLength: 2
 *                 maxLength: 100
 *                 description: 用户名称（2-100个字符）
 *                 example: John Doe
 *               password:
 *                 type: string
 *                 format: password
 *                 minLength: 8
 *                 maxLength: 128
 *                 description: 用户密码（至少8个字符，包含大小写字母和数字）
 *                 example: SecurePass123
 *     responses:
 *       201:
 *         description: 注册成功
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: true
 *                 data:
 *                   $ref: '#/components/schemas/AuthResponse'
 *                 message:
 *                   type: string
 *                   example: Registration successful
 *       400:
 *         description: 注册失败（验证错误或邮箱已存在）
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 *             examples:
 *               validation_error:
 *                 summary: 验证错误
 *                 value:
 *                   success: false
 *                   error:
 *                     code: VALIDATION_ERROR
 *                     message: 请提供有效的邮箱地址
 *               email_exists:
 *                 summary: 邮箱已存在
 *                 value:
 *                   success: false
 *                   error:
 *                     code: BAD_REQUEST
 *                     message: Email already exists
 */

/**
 * @swagger
 * /api/v1/auth/verify:
 *   post:
 *     summary: 验证 Token
 *     description: 验证 JWT Token 的有效性
 *     tags: [Authentication]
 *     security: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - token
 *             properties:
 *               token:
 *                 type: string
 *                 description: JWT Token
 *                 example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
 *     responses:
 *       200:
 *         description: Token 有效
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: true
 *                 data:
 *                   type: object
 *                   properties:
 *                     valid:
 *                       type: boolean
 *                       example: true
 *                     user:
 *                       $ref: '#/components/schemas/User'
 *       401:
 *         description: Token 无效或已过期
 */

/**
 * @swagger
 * /api/v1/auth/refresh:
 *   post:
 *     summary: 刷新 Token
 *     description: 使用当前有效的 Token 获取新的 JWT Token
 *     tags: [Authentication]
 *     security: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - token
 *             properties:
 *               token:
 *                 type: string
 *                 description: 当前有效的 JWT Token
 *     responses:
 *       200:
 *         description: Token 刷新成功
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *                   properties:
 *                     token:
 *                       type: string
 *                     expiresIn:
 *                       type: string
 *       401:
 *         description: Token 无效或已过期
 */

/**
 * @swagger
 * /api/v1/auth/logout:
 *   post:
 *     summary: 用户登出
 *     description: 用户登出，清除会话信息
 *     tags: [Authentication]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: 登出成功
 */

/**
 * @swagger
 * /api/v1/password/forgot:
 *   post:
 *     summary: 请求密码重置
 *     description: 请求发送密码重置邮件到用户邮箱
 *     tags: [Authentication]
 *     security: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - email
 *             properties:
 *               email:
 *                 type: string
 *                 format: email
 *                 description: 用户邮箱地址
 *                 example: user@example.com
 *     responses:
 *       200:
 *         description: 请求成功（如果邮箱存在，将发送重置邮件）
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: true
 *                 data:
 *                   type: object
 *                   properties:
 *                     message:
 *                       type: string
 *                       example: If email exists, reset link will be sent
 */

/**
 * @swagger
 * /api/v1/password/reset:
 *   post:
 *     summary: 重置密码
 *     description: 使用重置令牌设置新密码
 *     tags: [Authentication]
 *     security: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - token
 *               - newPassword
 *             properties:
 *               token:
 *                 type: string
 *                 minLength: 32
 *                 maxLength: 128
 *                 description: 重置令牌（从邮件链接获取）
 *               newPassword:
 *                 type: string
 *                 format: password
 *                 minLength: 8
 *                 maxLength: 128
 *                 description: 新密码（至少8个字符，包含大小写字母、数字和特殊字符）
 *     responses:
 *       200:
 *         description: 密码重置成功
 *       400:
 *         description: 重置令牌无效或已过期
 */

/**
 * @swagger
 * /api/v1/users:
 *   get:
 *     summary: 获取用户列表
 *     description: 获取所有用户列表（仅管理员可访问）
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *           default: 1
 *         description: 页码
 *       - in: query
 *         name: limit
 *         schema:
 *           type: integer
 *           default: 20
 *         description: 每页数量
 *     responses:
 *       200:
 *         description: 用户列表
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: array
 *                   items:
 *                     $ref: '#/components/schemas/User'
 *       401:
 *         description: 未授权
 *       403:
 *         description: 禁止访问（非管理员）
 *   post:
 *     summary: 创建用户
 *     description: 创建新用户账号（仅管理员可访问）
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - email
 *               - name
 *               - password
 *             properties:
 *               email:
 *                 type: string
 *                 format: email
 *               name:
 *                 type: string
 *                 minLength: 2
 *                 maxLength: 100
 *               password:
 *                 type: string
 *                 minLength: 8
 *                 maxLength: 128
 *     responses:
 *       201:
 *         description: 用户创建成功
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/User'
 *       400:
 *         description: 创建失败
 *       401:
 *         description: 未授权
 *       403:
 *         description: 禁止访问
 */

/**
 * @swagger
 * /api/v1/users/me:
 *   get:
 *     summary: 获取当前用户信息
 *     description: 获取已认证用户的详细信息
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: 用户信息
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   $ref: '#/components/schemas/User'
 *       401:
 *         description: 未授权
 *   put:
 *     summary: 更新当前用户信息
 *     description: 更新已认证用户的个人信息
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               email:
 *                 type: string
 *                 format: email
 *               name:
 *                 type: string
 *                 minLength: 2
 *                 maxLength: 100
 *               password:
 *                 type: string
 *                 minLength: 8
 *                 maxLength: 128
 *     responses:
 *       200:
 *         description: 更新成功
 *       400:
 *         description: 验证错误
 *       401:
 *         description: 未授权
 */

/**
 * @swagger
 * /api/v1/users/{id}:
 *   get:
 *     summary: 获取指定用户
 *     description: 根据 ID 获取用户信息（管理员可访问任意用户，普通用户仅可访问本人）
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: integer
 *         description: 用户 ID
 *     responses:
 *       200:
 *         description: 用户信息
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/User'
 *       401:
 *         description: 未授权
 *       403:
 *         description: 禁止访问
 *       404:
 *         description: 用户不存在
 *   put:
 *     summary: 更新指定用户
 *     description: 更新指定用户信息（管理员可更新任意用户，普通用户仅可更新本人非管理员字段）
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: integer
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               email:
 *                 type: string
 *                 format: email
 *               name:
 *                 type: string
 *               password:
 *                 type: string
 *               role:
 *                 type: string
 *                 enum: [user, admin, moderator]
 *     responses:
 *       200:
 *         description: 更新成功
 *       400:
 *         description: 验证错误
 *       401:
 *         description: 未授权
 *       403:
 *         description: 禁止访问
 *       404:
 *         description: 用户不存在
 *   delete:
 *     summary: 删除用户
 *     description: 删除指定用户（仅管理员可访问）
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: integer
 *     responses:
 *       204:
 *         description: 删除成功
 *       401:
 *         description: 未授权
 *       403:
 *         description: 禁止访问
 *       404:
 *         description: 用户不存在
 */

/**
 * @swagger
 * /api/v1/notifications:
 *   get:
 *     summary: 获取通知列表
 *     description: 获取当前用户的通知列表，支持分页和状态筛选
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *           default: 1
 *         description: 页码
 *       - in: query
 *         name: limit
 *         schema:
 *           type: integer
 *           default: 20
 *         description: 每页数量
 *       - in: query
 *         name: status
 *         schema:
 *           type: string
 *           enum: [unread, read, archived]
 *         description: 通知状态筛选
 *     responses:
 *       200:
 *         description: 通知列表
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: array
 *                   items:
 *                     $ref: '#/components/schemas/Notification'
 *       401:
 *         description: 未授权
 */

/**
 * @swagger
 * /api/v1/notifications/unread/count:
 *   get:
 *     summary: 获取未读通知数量
 *     description: 获取当前用户的未读通知数量
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: 未读数量
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   type: object
 *                   properties:
 *                     count:
 *                       type: integer
 *       401:
 *         description: 未授权
 */

/**
 * @swagger
 * /api/v1/notifications/{id}:
 *   get:
 *     summary: 获取通知详情
 *     description: 根据 ID 获取指定通知的详细信息
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *         description: 通知 ID
 *     responses:
 *       200:
 *         description: 通知详情
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Notification'
 *       401:
 *         description: 未授权
 *       404:
 *         description: 通知不存在
 *   delete:
 *     summary: 删除通知
 *     description: 删除指定通知
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: 删除成功
 *       401:
 *         description: 未授权
 *       404:
 *         description: 通知不存在
 */

/**
 * @swagger
 * /api/v1/notifications/{id}/read:
 *   put:
 *     summary: 标记通知为已读
 *     description: 将指定通知标记为已读状态
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: 操作成功
 *       401:
 *         description: 未授权
 *       404:
 *         description: 通知不存在
 */

/**
 * @swagger
 * /api/v1/notifications/read-all:
 *   put:
 *     summary: 标记所有通知为已读
 *     description: 将当前用户的所有通知标记为已读状态
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: 操作成功
 *       401:
 *         description: 未授权
 */

/**
 * @swagger
 * /api/v1/notifications/send:
 *   post:
 *     summary: 发送通知
 *     description: 向指定用户发送通知
 *     tags: [Notifications]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - userId
 *               - title
 *               - message
 *             properties:
 *               userId:
 *                 type: string
 *                 description: 目标用户 ID
 *               title:
 *                 type: string
 *                 description: 通知标题
 *               message:
 *                 type: string
 *                 description: 通知内容
 *               type:
 *                 type: string
 *                 enum: [system, user, security, promotion]
 *                 default: system
 *               channels:
 *                 type: array
 *                 items:
 *                   type: string
 *                   enum: [email, sms, push, in_app]
 *                 default: [in_app]
 *     responses:
 *       201:
 *         description: 发送成功
 *       400:
 *         description: 请求参数错误
 *       401:
 *         description: 未授权
 */

const options = {
  definition: {
    openapi: '3.0.0',
    info: {
      title: 'HJTPX API Documentation',
      version: '1.0.0',
      description: 'HJTPX 应用程序完整 API 文档。提供用户认证、用户管理、通知系统等功能。',
      termsOfService: 'https://hjtpx.com/terms',
      contact: {
        name: 'API Support',
        email: 'support@hjtpx.com',
        url: 'https://hjtpx.com/support'
      },
      license: {
        name: 'MIT',
        url: 'https://opensource.org/licenses/MIT'
      }
    },
    servers: [
      {
        url: 'http://localhost:3000',
        description: '开发环境服务器'
      },
      {
        url: 'https://api.hjtpx.com',
        description: '生产环境服务器'
      },
      {
        url: 'https://staging-api.hjtpx.com',
        description: '预发布环境服务器'
      }
    ],
    tags: [
      { name: 'Authentication', description: '用户认证接口 - 登录、注册、Token 管理' },
      { name: 'Users', description: '用户管理接口 - 用户信息的增删改查' },
      { name: 'Notifications', description: '通知管理接口 - 通知的发送、读取和管理' },
      { name: 'Health', description: '健康检查接口 - 服务状态监控' }
    ],
    components: {
      securitySchemes: {
        bearerAuth: {
          type: 'http',
          scheme: 'bearer',
          bearerFormat: 'JWT',
          description: '输入 JWT 访问令牌。登录成功后获取，有效期 7 天。'
        },
        apiKeyAuth: {
          type: 'apiKey',
          in: 'header',
          name: 'X-API-Key',
          description: '外部访问 API 密钥'
        }
      },
      schemas: {
        User: {
          type: 'object',
          properties: {
            id: { 
              type: 'integer', 
              description: '用户唯一标识符',
              example: 1
            },
            email: { 
              type: 'string', 
              format: 'email',
              description: '用户邮箱地址',
              example: 'user@example.com'
            },
            name: { 
              type: 'string',
              description: '用户显示名称',
              example: 'John Doe'
            },
            username: { 
              type: 'string',
              description: '用户名'
            },
            role: { 
              type: 'string', 
              enum: ['user', 'admin', 'moderator'],
              description: '用户角色',
              example: 'user'
            },
            createdAt: { 
              type: 'string', 
              format: 'date-time',
              description: '创建时间'
            },
            updatedAt: { 
              type: 'string', 
              format: 'date-time',
              description: '最后更新时间'
            }
          }
        },
        AuthResponse: {
          type: 'object',
          properties: {
            user: {
              $ref: '#/components/schemas/User'
            },
            token: {
              type: 'string',
              description: 'JWT 访问令牌'
            },
            expiresIn: {
              type: 'string',
              description: 'Token 过期时间',
              example: '7d'
            }
          }
        },
        Notification: {
          type: 'object',
          properties: {
            id: { 
              type: 'string', 
              format: 'uuid',
              description: '通知唯一标识符'
            },
            userId: { 
              type: 'string', 
              format: 'uuid',
              description: '关联用户 ID'
            },
            title: { 
              type: 'string',
              description: '通知标题'
            },
            message: { 
              type: 'string',
              description: '通知内容'
            },
            type: { 
              type: 'string', 
              enum: ['system', 'user', 'security', 'promotion'],
              description: '通知类型'
            },
            status: { 
              type: 'string', 
              enum: ['unread', 'read', 'archived'],
              description: '通知状态'
            },
            channels: {
              type: 'array',
              items: { 
                type: 'string', 
                enum: ['email', 'sms', 'push', 'in_app'] 
              },
              description: '通知渠道列表'
            },
            createdAt: { 
              type: 'string', 
              format: 'date-time'
            }
          }
        },
        HealthCheck: {
          type: 'object',
          properties: {
            status: {
              type: 'string',
              enum: ['healthy', 'unhealthy', 'unavailable'],
              description: '健康状态'
            },
            message: {
              type: 'string',
              description: '状态描述信息'
            },
            responseTime: {
              type: 'string',
              description: '响应时间'
            },
            usage: {
              type: 'object',
              description: '资源使用情况（内存）'
            },
            loadAverage: {
              type: 'array',
              items: { type: 'number' },
              description: 'CPU 负载平均值'
            }
          }
        },
        Error: {
          type: 'object',
          properties: {
            success: { 
              type: 'boolean', 
              default: false,
              example: false
            },
            error: {
              type: 'object',
              properties: {
                message: { 
                  type: 'string',
                  example: 'Error message'
                },
                code: { 
                  type: 'string',
                  example: 'ERROR_CODE'
                },
                details: {
                  type: 'string',
                  description: '详细错误信息'
                }
              }
            },
            timestamp: {
              type: 'string',
              format: 'date-time'
            }
          }
        },
        SuccessResponse: {
          type: 'object',
          properties: {
            success: { 
              type: 'boolean', 
              default: true,
              example: true
            },
            data: { 
              type: 'object',
              description: '响应数据'
            },
            message: { 
              type: 'string',
              example: 'Operation successful'
            },
            timestamp: {
              type: 'string',
              format: 'date-time'
            }
          }
        },
        PaginationMeta: {
          type: 'object',
          properties: {
            page: { type: 'integer' },
            limit: { type: 'integer' },
            total: { type: 'integer' },
            pages: { type: 'integer' }
          }
        }
      },
      responses: {
        Unauthorized: {
          description: '未授权 - 认证失败或 Token 无效',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' },
              example: {
                success: false,
                error: {
                  code: 'UNAUTHORIZED',
                  message: 'Invalid or expired token'
                }
              }
            }
          }
        },
        Forbidden: {
          description: '禁止访问 - 权限不足',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' },
              example: {
                success: false,
                error: {
                  code: 'FORBIDDEN',
                  message: 'Access denied'
                }
              }
            }
          }
        },
        NotFound: {
          description: '未找到 - 资源不存在',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' },
              example: {
                success: false,
                error: {
                  code: 'NOT_FOUND',
                  message: 'Resource not found'
                }
              }
            }
          }
        },
        ValidationError: {
          description: '验证错误 - 输入数据格式不正确',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' },
              example: {
                success: false,
                error: {
                  code: 'VALIDATION_ERROR',
                  message: 'Invalid input data',
                  details: [
                    {
                      field: 'email',
                      message: '请提供有效的邮箱地址',
                      type: 'string.email'
                    }
                  ]
                }
              }
            }
          }
        },
        TooManyRequests: {
          description: '请求过于频繁 - 触发限流',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' },
              example: {
                success: false,
                error: {
                  code: 'TOO_MANY_REQUESTS',
                  message: 'Too many requests, please try again later.',
                  retryAfter: 60
                }
              }
            }
          }
        },
        InternalServerError: {
          description: '服务器内部错误',
          content: {
            'application/json': {
              schema: { $ref: '#/components/schemas/Error' },
              example: {
                success: false,
                error: {
                  code: 'INTERNAL_ERROR',
                  message: 'An unexpected error occurred'
                }
              }
            }
          }
        }
      }
    },
    security: [
      { bearerAuth: [] }
    ]
  },
  apis: [
    './src/backend/routes/*.js',
    './src/backend/routes/v1/*.js'
  ]
};

const swaggerSpec = swaggerJsdoc(options);

module.exports = swaggerSpec;

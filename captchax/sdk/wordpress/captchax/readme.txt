=== CaptchaX - 行为验证码 ===
Contributors: captchax
Tags: captcha, verification, security, anti-spam, login, comment, register
Requires at least: 5.0
Tested up to: 6.4
Requires PHP: 5.6
Stable tag: 1.0.0
License: GPLv2 or later
License URI: https://www.gnu.org/licenses/gpl-2.0.html

CaptchaX 现代化行为验证码系统，支持滑块、点选、拼图等多种验证方式。

== Description ==

CaptchaX 是一个功能强大的 WordPress 验证码插件，提供以下功能：

* **多种验证方式**: 支持滑块验证、点选验证、拼图验证、旋转验证、文字验证
* **表单保护**: 自动保护评论、登录、注册、找回密码等表单
* **主题切换**: 支持浅色和深色主题
* **简单配置**: 直观的设置界面，快速上手
* **安全可靠**: 基于 Token 的安全验证机制

= 支持的表单 =

* 评论表单
* 登录表单
* 注册表单
* 找回密码表单

= 技术特性 =

* PHP 5.6+ 兼容
* WordPress 5.0+ 兼容
* 国际化支持（i18n）
* 安全过滤（sanitize）
* nonce 验证

== Installation ==

1. 上传 `captchax` 文件夹到 `/wp-content/plugins/` 目录
2. 在 WordPress 的'插件'菜单中激活插件
3. 在设置页面配置您的 API Key 和 API Secret
4. 选择需要保护的表单和验证码类型

== Frequently Asked Questions ==

= 如何获取 API Key？ =

访问 [CaptchaX 官网](https://captchax.com) 注册账号并获取 API Key。

= 支持哪些验证类型？ =

当前支持：滑块验证、点选验证、拼图验证、旋转验证、文字验证。

= 如何自定义主题？ =

在设置页面选择浅色或深色主题。

== Changelog ==

= 1.0.0 =
* 初始版本发布
* 支持评论、登录、注册、找回密码表单
* 支持多种验证码类型
* 支持浅色和深色主题

== Upgrade Notice ==

= 1.0.0 =
初始版本发布

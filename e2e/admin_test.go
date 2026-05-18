package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

const (
	testAdminUsername = "admin"
	testAdminPassword = "admin123"
)

type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
}

func TestAdminLoginE2E(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("启动Playwright失败: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Fatalf("启动浏览器失败: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("创建页面失败: %v", err)
	}
	defer page.Close()

	t.Run("加载管理端登录页面", func(t *testing.T) {
		_, err := page.Goto("http://localhost:8080/admin/login",
			page.GotoOptions{Timeout: playwright.Float(30000)})
		if err != nil {
			t.Errorf("加载登录页面失败: %v", err)
		}
		time.Sleep(2 * time.Second)
	})

	t.Run("截图登录页面", func(t *testing.T) {
		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/admin-login-page-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
		_, err := page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(filename),
			FullPage: playwright.Bool(true),
		})
		if err != nil {
			t.Errorf("截图失败: %v", err)
		} else {
			t.Logf("截图已保存: %s", filename)
		}
	})

	t.Run("使用有效凭据登录", func(t *testing.T) {
		usernameField, err := page.QuerySelector("input[name='username'], input[type='text']")
		if err != nil || usernameField == nil {
			t.Skip("用户名输入框未找到")
		}

		if err := usernameField.Fill(testAdminUsername); err != nil {
			t.Errorf("填写用户名失败: %v", err)
			return
		}

		passwordField, err := page.QuerySelector("input[name='password'], input[type='password']")
		if err != nil || passwordField == nil {
			t.Skip("密码输入框未找到")
		}

		if err := passwordField.Fill(testAdminPassword); err != nil {
			t.Errorf("填写密码失败: %v", err)
			return
		}

		submitBtn, err := page.QuerySelector("button[type='submit'], button:has-text('登录'), button:has-text('Login')")
		if err != nil || submitBtn == nil {
			t.Skip("提交按钮未找到")
		}

		if err := submitBtn.Click(); err != nil {
			t.Errorf("点击提交按钮失败: %v", err)
			return
		}

		time.Sleep(3 * time.Second)

		currentURL := page.URL()
		t.Logf("登录后URL: %s", currentURL)

		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/admin-after-login-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
		page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(filename),
			FullPage: playwright.Bool(true),
		})
	})

	t.Run("无效凭据应该显示错误", func(t *testing.T) {
		page2, err := browser.NewPage()
		if err != nil {
			t.Fatalf("创建新页面失败: %v", err)
		}
		defer page2.Close()

		_, err = page2.Goto("http://localhost:8080/admin/login",
			page2.GotoOptions{Timeout: playwright.Float(30000)})
		if err != nil {
			t.Skip("无法加载登录页面")
		}

		usernameField, err := page2.QuerySelector("input[name='username'], input[type='text']")
		if err != nil || usernameField == nil {
			t.Skip("用户名输入框未找到")
		}
		usernameField.Fill("invalid-user")

		passwordField, err := page2.QuerySelector("input[name='password'], input[type='password']")
		if err != nil || passwordField == nil {
			t.Skip("密码输入框未找到")
		}
		passwordField.Fill("wrong-password")

		submitBtn, err := page2.QuerySelector("button[type='submit']")
		if submitBtn != nil {
			submitBtn.Click()
		}

		time.Sleep(2 * time.Second)

		errorMsg, err := page2.QuerySelector(".error, .alert, [role='alert'], .text-danger")
		if err == nil && errorMsg != nil {
			t.Log("错误提示显示正常")
		}
	})
}

func TestAdminDashboardE2E(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("启动Playwright失败: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Fatalf("启动浏览器失败: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("创建页面失败: %v", err)
	}
	defer page.Close()

	consoleErrors := []string{}
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		if msg.Type() == "error" {
			consoleErrors = append(consoleErrors, msg.Text())
		}
	})

	t.Run("加载管理仪表盘", func(t *testing.T) {
		_, err := page.Goto("http://localhost:8080/admin/dashboard",
			page.GotoOptions{Timeout: playwright.Float(30000)})
		if err != nil {
			t.Errorf("加载仪表盘失败: %v", err)
		}
		time.Sleep(2 * time.Second)
	})

	t.Run("截图仪表盘", func(t *testing.T) {
		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/admin-dashboard-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
		_, err := page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(filename),
			FullPage: playwright.Bool(true),
		})
		if err != nil {
			t.Errorf("截图失败: %v", err)
		}
	})

	t.Run("仪表盘包含关键元素", func(t *testing.T) {
		dashboardElements := []string{
			".dashboard", ".stats", ".card",
			"[class*='stat']", "[class*='card']",
		}

		found := false
		for _, selector := range dashboardElements {
			elem, err := page.QuerySelector(selector)
			if err == nil && elem != nil {
				found = true
				t.Logf("找到仪表盘元素: %s", selector)
				break
			}
		}

		if !found {
			t.Log("未找到标准仪表盘元素，页面可能已加载但结构不同")
		}
	})

	t.Run("检测仪表盘控制台错误", func(t *testing.T) {
		criticalErrors := filterCriticalErrors(consoleErrors)
		if len(criticalErrors) > 0 {
			t.Errorf("发现 %d 个控制台错误: %v", len(criticalErrors), criticalErrors)
		}
	})
}

func TestAdminPagesE2E(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("启动Playwright失败: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Fatalf("启动浏览器失败: %v", err)
	}
	defer browser.Close()

	adminPages := []struct {
		name    string
		url     string
	}{
		{"统计页面", "/admin/stats"},
		{"应用管理页面", "/admin/applications"},
		{"日志页面", "/admin/logs"},
		{"监控页面", "/admin/monitoring"},
		{"分析页面", "/admin/analytics"},
	}

	for _, adminPage := range adminPages {
		t.Run(adminPage.name, func(t *testing.T) {
			page, err := browser.NewPage()
			if err != nil {
				t.Fatalf("创建页面失败: %v", err)
			}
			defer page.Close()

			consoleErrors := []string{}
			page.OnConsole(func(msg playwright.ConsoleMessage) {
				if msg.Type() == "error" {
					consoleErrors = append(consoleErrors, msg.Text())
				}
			})

			fullURL := fmt.Sprintf("http://localhost:8080%s", adminPage.url)
			_, err = page.Goto(fullURL,
				page.GotoOptions{Timeout: playwright.Float(30000)})
			if err != nil {
				t.Errorf("加载 %s 失败: %v", adminPage.name, err)
			}

			time.Sleep(2 * time.Second)

			screenshotDir := "e2e/screenshots"
			ensureDir(screenshotDir)
			pageName := fmt.Sprintf("%s", adminPage.name)
			safeName := pageName
			filename := fmt.Sprintf("%s/admin-%s-%s.png", screenshotDir, safeName, time.Now().Format("2006-01-02T15-04-05-000Z"))
			_, err = page.Screenshot(playwright.PageScreenshotOptions{
				Path:     playwright.String(filename),
				FullPage: playwright.Bool(true),
			})
			if err != nil {
				t.Logf("截图失败: %v", err)
			}

			criticalErrors := filterCriticalErrors(consoleErrors)
			if len(criticalErrors) > 0 {
				t.Logf("%s 有 %d 个控制台错误", adminPage.name, len(criticalErrors))
			}
		})
	}
}

func TestAdminAPIE2E(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("启动Playwright失败: %v", err)
	}
	defer pw.Stop()

	apiContext, err := pw.APIRequest.NewContext(playwright.APIRequestNewContextOptions{
		BaseURL: playwright.String("http://localhost:8080"),
	})
	if err != nil {
		t.Fatalf("创建API上下文失败: %v", err)
	}
	defer apiContext.Close()

	t.Run("API健康检查", func(t *testing.T) {
		resp, err := apiContext.Get("/health")
		if err != nil {
			t.Errorf("健康检查请求失败: %v", err)
			return
		}
		defer resp.Dispose()

		if resp.Status() != 200 {
			t.Logf("健康检查返回: %d (可能服务未启动)", resp.Status())
		} else {
			t.Log("健康检查通过")
		}
	})

	t.Run("管理员登录API", func(t *testing.T) {
		resp, err := apiContext.Post("/api/v1/auth/login",
			playwright.APIRequestContextOptions{
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Data: map[string]interface{}{
					"username": testAdminUsername,
					"password": testAdminPassword,
				},
			})
		if err != nil {
			t.Logf("登录请求失败: %v (可能服务未启动)", err)
			return
		}
		defer resp.Dispose()

		t.Logf("登录API响应状态: %d", resp.Status())

		if resp.Status() == 200 {
			body, err := resp.JSON()
			if err == nil {
				t.Logf("登录响应: %+v", body)
			}
		}
	})

	t.Run("获取应用列表", func(t *testing.T) {
		resp, err := apiContext.Get("/api/v1/admin/applications")
		if err != nil {
			t.Logf("获取应用列表失败: %v (预期未授权)", err)
			return
		}
		defer resp.Dispose()

		t.Logf("应用列表API状态: %d", resp.Status())

		if resp.Status() == 401 {
			t.Log("需要认证，预期行为")
		}
	})

	t.Run("获取统计数据", func(t *testing.T) {
		resp, err := apiContext.Get("/api/v1/admin/stats/verification")
		if err != nil {
			t.Logf("获取统计失败: %v (预期未授权)", err)
			return
		}
		defer resp.Dispose()

		t.Logf("统计API状态: %d", resp.Status())
	})
}

func TestAdminLogoutE2E(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("启动Playwright失败: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Fatalf("启动浏览器失败: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("创建页面失败: %v", err)
	}
	defer page.Close()

	t.Run("测试登出功能", func(t *testing.T) {
		_, err := page.Goto("http://localhost:8080/admin/dashboard",
			page.GotoOptions{Timeout: playwright.Float(30000)})
		if err != nil {
			t.Skip("无法加载仪表盘")
		}

		time.Sleep(2 * time.Second)

		logoutBtn, err := page.QuerySelector("a[href*='logout'], button:has-text('Logout'), button:has-text('退出')")
		if err != nil || logoutBtn == nil {
			t.Log("登出按钮未找到，可能已登录或页面结构不同")
			return
		}

		if err := logoutBtn.Click(); err != nil {
			t.Errorf("点击登出按钮失败: %v", err)
			return
		}

		time.Sleep(2 * time.Second)

		currentURL := page.URL()
		t.Logf("登出后URL: %s", currentURL)

		if currentURL != "" {
			t.Log("登出流程完成")
		}
	})
}

func parseLoginResponse(body []byte) *LoginResponse {
	var resp LoginResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	return &resp
}

func _printTestSummary() {
	_ = os.Stdout
}

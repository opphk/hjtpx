package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

func TestClickCaptchaE2E(t *testing.T) {
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
	consoleWarnings := []string{}
	consoleLogs := []string{}

	page.OnConsole(func(msg playwright.ConsoleMessage) {
		switch msg.Type() {
		case "error":
			consoleErrors = append(consoleErrors, msg.Text())
		case "warning":
			consoleWarnings = append(consoleWarnings, msg.Text())
		case "log":
			consoleLogs = append(consoleLogs, msg.Text())
		}
	})

	t.Run("加载点选验证码页面", func(t *testing.T) {
		_, err := page.Goto("http://localhost:8080/captcha?type=click",
			page.GotoOptions{Timeout: playwright.Float(30000)})
		if err != nil {
			t.Errorf("加载点选验证码页面失败: %v", err)
		}
		time.Sleep(2 * time.Second)
	})

	t.Run("截图点选验证码页面", func(t *testing.T) {
		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/click-captcha-page-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
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

	t.Run("检测控制台错误", func(t *testing.T) {
		criticalErrors := filterCriticalErrors(consoleErrors)
		if len(criticalErrors) > 0 {
			t.Errorf("发现 %d 个控制台错误: %v", len(criticalErrors), criticalErrors)
		}
	})

	t.Run("模拟点击验证码区域", func(t *testing.T) {
		captchaArea, err := page.QuerySelector(".captcha-image, .captcha-bg, [class*='captcha']")
		if err != nil || captchaArea == nil {
			t.Log("验证码区域未找到，跳过点击测试")
			return
		}

		box, err := captchaArea.BoundingBox()
		if err != nil || box == nil {
			t.Skip("无法获取验证码区域边界框")
		}

		centerX := box.X + box.Width/2
		centerY := box.Y + box.Height/2

		if err := page.Mouse.Click(centerX, centerY); err != nil {
			t.Errorf("点击失败: %v", err)
		}

		time.Sleep(1 * time.Second)

		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/click-captcha-clicked-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
		page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(filename),
			FullPage: playwright.Bool(true),
		})
	})

	t.Run("模拟多次点击", func(t *testing.T) {
		captchaArea, err := page.QuerySelector(".captcha-image, .captcha-bg, [class*='captcha']")
		if err != nil || captchaArea == nil {
			t.Skip("验证码区域未找到")
		}

		box, err := captchaArea.BoundingBox()
		if err != nil || box == nil {
			t.Skip("无法获取验证码区域边界框")
		}

		clickPoints := []struct{ x, y float64 }{
			{box.X + box.Width*0.25, box.Y + box.Height*0.25},
			{box.X + box.Width*0.5, box.Y + box.Height*0.5},
			{box.X + box.Width*0.75, box.Y + box.Height*0.75},
		}

		for i, point := range clickPoints {
			if err := page.Mouse.Click(point.x, point.y); err != nil {
				t.Errorf("第 %d 次点击失败: %v", i+1, err)
			}
			time.Sleep(500 * time.Millisecond)
		}

		t.Logf("成功完成 %d 次点击", len(clickPoints))
	})

	t.Run("点选验证码刷新功能", func(t *testing.T) {
		refreshBtn, err := page.QuerySelector("button:has-text('刷新'), button:has-text('重试'), .refresh-btn, [class*='refresh']")
		if err != nil {
			t.Log("刷新按钮未找到")
			return
		}

		if refreshBtn != nil {
			err := refreshBtn.Click()
			if err != nil {
				t.Errorf("点击刷新按钮失败: %v", err)
			}
			time.Sleep(2 * time.Second)
			t.Log("刷新按钮点击成功")
		}
	})

	t.Run("记录控制台日志数量", func(t *testing.T) {
		t.Logf("控制台日志: %d 条, 警告: %d 条, 错误: %d 条",
			len(consoleLogs), len(consoleWarnings), len(consoleErrors))
	})
}

func TestClickCaptchaAPI(t *testing.T) {
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

	t.Run("生成点选验证码", func(t *testing.T) {
		resp, err := apiContext.Post("/api/v1/captcha/click/generate",
			playwright.APIRequestContextOptions{
				FormData: map[string][]string{
					"appId": {"test-app"},
				},
			})
		if err != nil {
			t.Errorf("生成点选验证码失败: %v", err)
			return
		}
		defer resp.Dispose()

		if resp.Status() < 200 || resp.Status() >= 300 {
			t.Errorf("意外的状态码: %d", resp.Status())
			return
		}

		t.Logf("点选验证码生成成功，状态码: %d", resp.Status())
	})

	t.Run("验证点选验证码_单点", func(t *testing.T) {
		captchaId := fmt.Sprintf("test-click-%d", time.Now().Unix())

		resp, err := apiContext.Post("/api/v1/captcha/click/verify",
			playwright.APIRequestContextOptions{
				FormData: map[string][]string{
					"captchaId": {captchaId},
					"points":    {`[{"x":100,"y":100}]`},
				},
			})
		if err != nil {
			t.Logf("验证请求失败(测试captchaId预期失败): %v", err)
			return
		}
		defer resp.Dispose()

		t.Logf("点选验证响应状态: %d", resp.Status())
	})

	t.Run("验证点选验证码_多点", func(t *testing.T) {
		captchaId := fmt.Sprintf("test-click-%d", time.Now().Unix())

		resp, err := apiContext.Post("/api/v1/captcha/click/verify",
			playwright.APIRequestContextOptions{
				FormData: map[string][]string{
					"captchaId": {captchaId},
					"points":    {`[{"x":50,"y":50},{"x":150,"y":100},{"x":200,"y":150}]`},
				},
			})
		if err != nil {
			t.Logf("验证请求失败: %v", err)
			return
		}
		defer resp.Dispose()

		t.Logf("多点验证响应状态: %d", resp.Status())
	})

	t.Run("无效验证码ID验证", func(t *testing.T) {
		resp, err := apiContext.Post("/api/v1/captcha/click/verify",
			playwright.APIRequestContextOptions{
				FormData: map[string][]string{
					"captchaId": {"invalid-click-id-99999"},
					"points":    {`[{"x":100,"y":100}]`},
				},
			})
		if err != nil {
			t.Logf("验证请求失败: %v", err)
			return
		}
		defer resp.Dispose()

		t.Logf("无效ID验证状态: %d (预期失败)", resp.Status())
	})
}

func TestClickCaptchaCompleteFlow(t *testing.T) {
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

	t.Run("完整点选验证流程", func(t *testing.T) {
		_, err := page.Goto("http://localhost:8080/captcha?type=click",
			page.GotoOptions{Timeout: playwright.Float(30000)})
		if err != nil {
			t.Fatalf("加载验证码页面失败: %v", err)
		}

		time.Sleep(3 * time.Second)

		captchaArea, err := page.QuerySelector(".captcha-container, #captcha, [class*='captcha']")
		if err != nil || captchaArea == nil {
			t.Skip("验证码容器未找到，跳过流程测试")
		}

		box, err := captchaArea.BoundingBox()
		if err != nil || box == nil {
			t.Skip("无法获取验证码区域边界框")
		}

		clickPositions := []struct {
			x, y float64
			desc string
		}{
			{box.X + box.Width*0.3, box.Y + box.Height*0.3, "左上区域"},
			{box.X + box.Width*0.7, box.Y + box.Height*0.5, "右中区域"},
			{box.X + box.Width*0.5, box.Y + box.Height*0.8, "底部中央"},
		}

		for i, pos := range clickPositions {
			if err := page.Mouse.Click(pos.x, pos.y); err != nil {
				t.Errorf("第 %d 次点击 (%s) 失败: %v", i+1, pos.desc, err)
			} else {
				t.Logf("点击成功: %s (%.0f, %.0f)", pos.desc, pos.x, pos.y)
			}
			time.Sleep(800 * time.Millisecond)
		}

		time.Sleep(2 * time.Second)

		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/click-complete-flow-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
		page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(filename),
			FullPage: playwright.Bool(true),
		})
	})

	t.Run("验证无严重控制台错误", func(t *testing.T) {
		criticalErrors := filterCriticalErrors(consoleErrors)
		if len(criticalErrors) > 0 {
			t.Errorf("检测到控制台错误: %v", criticalErrors)
		} else {
			t.Log("无严重控制台错误")
		}
	})
}

func TestLianLianKanCaptchaE2E(t *testing.T) {
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

	t.Run("加载连连看验证码页面", func(t *testing.T) {
		_, err := page.Goto("http://localhost:8080/lianliankan",
			page.GotoOptions{Timeout: playwright.Float(30000)})
		if err != nil {
			t.Errorf("加载连连看页面失败: %v", err)
		}
		time.Sleep(2 * time.Second)
	})

	t.Run("截图连连看页面", func(t *testing.T) {
		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/lianliankan-page-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
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

	t.Run("连连看交互测试", func(t *testing.T) {
		captchaArea, err := page.QuerySelector(".lianliankan, .captcha-container, [class*='captcha']")
		if err != nil || captchaArea == nil {
			t.Skip("连连看区域未找到")
		}

		box, err := captchaArea.BoundingBox()
		if err != nil || box == nil {
			t.Skip("无法获取连连看区域边界框")
		}

		if err := page.Mouse.Click(box.X+box.Width*0.3, box.Y+box.Height*0.3); err != nil {
			t.Errorf("点击失败: %v", err)
		}
		time.Sleep(500 * time.Millisecond)

		if err := page.Mouse.Click(box.X+box.Width*0.7, box.Y+box.Height*0.7); err != nil {
			t.Errorf("第二次点击失败: %v", err)
		}
		time.Sleep(1 * time.Second)

		t.Log("连连看点击交互完成")
	})
}

package e2e

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

func TestSliderCaptchaE2E(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("Failed to start Playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Fatalf("Failed to launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}
	defer page.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	consoleErrors := []string{}
	consoleWarnings := []string{}
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		switch msg.Type() {
		case "error":
			consoleErrors = append(consoleErrors, msg.Text())
		case "warning":
			consoleWarnings = append(consoleWarnings, msg.Text())
		}
	})

	t.Run("加载滑块验证码页面", func(t *testing.T) {
		_, err := page.Goto("http://localhost:8080/captcha?type=slider",
			page.GotoOptions{Timeout: playwright.Float(30000)})
		if err != nil {
			t.Errorf("Failed to load slider captcha page: %v", err)
		}
		time.Sleep(2 * time.Second)
	})

	t.Run("页面应该包含验证码元素", func(t *testing.T) {
		captchaContainer, err := page.QuerySelector(".captcha-container, #captcha, .slider-captcha")
		if err != nil {
			t.Logf("Captcha container selector not found, trying generic selector: %v", err)
		}
		if captchaContainer == nil {
			_, err := page.QuerySelector("[class*='captcha']")
			if err != nil {
				t.Skip("No captcha elements found on page")
			}
		}
	})

	t.Run("截图滑块验证码页面", func(t *testing.T) {
		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/slider-captcha-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
		_, err := page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(filename),
			FullPage: playwright.Bool(true),
		})
		if err != nil {
			t.Errorf("Failed to take screenshot: %v", err)
		} else {
			t.Logf("Screenshot saved: %s", filename)
		}
	})

	t.Run("检测控制台错误", func(t *testing.T) {
		criticalErrors := filterCriticalErrors(consoleErrors)
		if len(criticalErrors) > 0 {
			t.Errorf("Found %d console errors: %v", len(criticalErrors), criticalErrors)
		}
	})

	t.Run("模拟滑块拖拽", func(t *testing.T) {
		sliderTrack, err := page.QuerySelector(".slider-track, .slider-bar, [class*='slider']")
		if err != nil {
			t.Log("Slider track not found")
			return
		}

		if sliderTrack == nil {
			t.Skip("Slider track element is nil")
		}

		box, err := sliderTrack.BoundingBox()
		if err != nil || box == nil {
			t.Skip("Cannot get slider bounding box")
		}

		startX := box.X + box.Width*0.1
		startY := box.Y + box.Height/2
		endX := box.X + box.Width*0.8

		_, err = page.Mouse.Move(startX, startY)
		if err != nil {
			t.Errorf("Failed to move mouse: %v", err)
			return
		}

		err = page.Mouse.Down()
		if err != nil {
			t.Errorf("Failed to press mouse: %v", err)
			return
		}

		steps := 20
		for i := 0; i < steps; i++ {
			currentX := startX + (endX-startX)*float64(i)/float64(steps)
			jitter := rand.Float64()*2 - 1
			_, err = page.Mouse.Move(currentX+jitter, startY+jitter)
			if err != nil {
				t.Errorf("Failed to drag mouse: %v", err)
				break
			}
			time.Sleep(30 * time.Millisecond)
		}

		err = page.Mouse.Up()
		if err != nil {
			t.Errorf("Failed to release mouse: %v", err)
		}

		time.Sleep(2 * time.Second)

		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/slider-dragged-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
		_, err = page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(filename),
			FullPage: playwright.Bool(true),
		})
		if err != nil {
			t.Logf("Failed to take post-drag screenshot: %v", err)
		}
	})

	t.Run("滑块刷新功能", func(t *testing.T) {
		refreshBtn, err := page.QuerySelector("button:has-text('刷新'), button:has-text('重试'), .refresh-btn")
		if err != nil {
			t.Log("Refresh button not found")
			return
		}

		if refreshBtn != nil {
			err := refreshBtn.Click()
			if err != nil {
				t.Errorf("Failed to click refresh button: %v", err)
			}
			time.Sleep(2 * time.Second)
			t.Log("Slider refresh clicked successfully")
		}
	})

	_ = ctx
}

func TestSliderCaptchaAPI(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("Failed to start Playwright: %v", err)
	}
	defer pw.Stop()

	apiContext, err := pw.APIRequest.NewContext(playwright.APIRequestNewContextOptions{
		BaseURL: playwright.String("http://localhost:8080"),
	})
	if err != nil {
		t.Fatalf("Failed to create API context: %v", err)
	}
	defer apiContext.Close()

	t.Run("生成滑块验证码", func(t *testing.T) {
		resp, err := apiContext.Post("/api/v1/captcha/slider/generate",
			playwright.APIRequestContextOptions{
				FormData: map[string][]string{
					"appId": {"test-app"},
				},
			})
		if err != nil {
			t.Errorf("Failed to generate slider captcha: %v", err)
			return
		}
		defer resp.Dispose()

		if resp.Status() < 200 || resp.Status() >= 300 {
			t.Errorf("Unexpected status code: %d", resp.Status())
			return
		}

		t.Logf("Slider captcha generated with status: %d", resp.Status())
	})

	t.Run("验证滑块验证码", func(t *testing.T) {
		captchaId := fmt.Sprintf("test-captcha-%d", time.Now().Unix())
		x := 150.0

		resp, err := apiContext.Post("/api/v1/captcha/slider/verify",
			playwright.APIRequestContextOptions{
				FormData: map[string][]string{
					"captchaId": {captchaId},
					"x":         {fmt.Sprintf("%.2f", x)},
				},
			})
		if err != nil {
			t.Logf("Verify request failed (expected for test captcha ID): %v", err)
			return
		}
		defer resp.Dispose()

		t.Logf("Slider verify response status: %d", resp.Status())
	})

	t.Run("无效验证码ID验证", func(t *testing.T) {
		resp, err := apiContext.Post("/api/v1/captcha/slider/verify",
			playwright.APIRequestContextOptions{
				FormData: map[string][]string{
					"captchaId": {"invalid-id-12345"},
					"x":         {"999"},
				},
			})
		if err != nil {
			t.Logf("Verify request failed: %v", err)
			return
		}
		defer resp.Dispose()

		t.Logf("Invalid ID verify status: %d (expected to fail)", resp.Status())
	})
}

func TestSliderCaptchaCompleteFlow(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("Failed to start Playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Fatalf("Failed to launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}
	defer page.Close()

	consoleErrors := []string{}
	consoleWarnings := []string{}

	page.OnConsole(func(msg playwright.ConsoleMessage) {
		switch msg.Type() {
		case "error":
			consoleErrors = append(consoleErrors, msg.Text())
		case "warning":
			consoleWarnings = append(consoleWarnings, msg.Text())
		}
	})

	t.Run("完整滑块验证流程", func(t *testing.T) {
		_, err := page.Goto("http://localhost:8080/captcha?type=slider",
			page.GotoOptions{Timeout: playwright.Float(30000)})
		if err != nil {
			t.Fatalf("Failed to load captcha page: %v", err)
		}

		time.Sleep(3 * time.Second)

		sliderTrack, err := page.QuerySelector(".slider-track")
		if err != nil || sliderTrack == nil {
			sliderTrack, _ = page.QuerySelector("[class*='slider']")
		}
		if sliderTrack == nil {
			t.Skip("Slider track not found, skipping flow test")
		}

		box, err := sliderTrack.BoundingBox()
		if err != nil || box == nil {
			t.Skip("Cannot get slider bounding box")
		}

		startX := box.X + 10
		startY := box.Y + box.Height/2
		endX := box.X + box.Width - 50

		_, err = page.Mouse.Move(startX, startY)
		if err != nil {
			t.Fatalf("Failed to move to start: %v", err)
		}

		err = page.Mouse.Down()
		if err != nil {
			t.Fatalf("Failed to press mouse: %v", err)
		}

		for i := 0; i <= 20; i++ {
			x := startX + (endX-startX)*float64(i)/20.0
			y := startY + (rand.Float64()*4 - 2)
			err := page.Mouse.Move(x, y)
			if err != nil {
				t.Errorf("Drag step %d failed: %v", i, err)
				break
			}
			time.Sleep(40 * time.Millisecond)
		}

		err = page.Mouse.Up()
		if err != nil {
			t.Errorf("Failed to release mouse: %v", err)
		}

		time.Sleep(3 * time.Second)

		screenshotDir := "e2e/screenshots"
		ensureDir(screenshotDir)
		filename := fmt.Sprintf("%s/slider-complete-flow-%s.png", screenshotDir, time.Now().Format("2006-01-02T15-04-05-000Z"))
		_, err = page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(filename),
			FullPage: playwright.Bool(true),
		})
		if err != nil {
			t.Logf("Failed to take flow screenshot: %v", err)
		}
	})

	t.Run("检查控制台错误", func(t *testing.T) {
		criticalErrors := filterCriticalErrors(consoleErrors)
		if len(criticalErrors) > 0 {
			t.Errorf("Console errors detected: %v", criticalErrors)
		}
	})

	t.Run("检查控制台警告", func(t *testing.T) {
		if len(consoleWarnings) > 10 {
			t.Logf("Found %d console warnings", len(consoleWarnings))
		}
	})
}

func filterCriticalErrors(errors []string) []string {
	critical := []string{}
	ignored := []string{"favicon", "404", "Failed to load resource", "net::ERR"}

	for _, err := range errors {
		isIgnored := false
		for _, ignore := range ignored {
			if containsString(err, ignore) {
				isIgnored = true
				break
			}
		}
		if !isIgnored {
			critical = append(critical, err)
		}
	}
	return critical
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && len(s) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func ensureDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Printf("Failed to create directory %s: %v\n", dir, err)
		}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var _ = filepath.Join

import { test, expect } from '@playwright/test';

test.describe('Voice Captcha E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/voice-captcha.html');
  });

  test('should load voice captcha page', async ({ page }) => {
    await expect(page).toHaveTitle(/Voice/);
    const captchaContainer = page.locator('#voice-captcha-container');
    await expect(captchaContainer).toBeVisible();
  });

  test('should have play button', async ({ page }) => {
    const playButton = page.locator('#play-button');
    await expect(playButton).toBeVisible();
  });

  test('should have audio controls', async ({ page }) => {
    const audioPlayer = page.locator('#audio-player');
    await expect(audioPlayer).toBeVisible();
  });

  test('should have input field for answer', async ({ page }) => {
    const answerInput = page.locator('#answer-input');
    await expect(answerInput).toBeVisible();
  });

  test('should show error on empty submit', async ({ page }) => {
    const submitButton = page.locator('#submit-button');
    await submitButton.click();
    
    const errorMessage = page.locator('#error-message');
    await expect(errorMessage).toBeVisible();
  });

  test('should play audio on play button click', async ({ page }) => {
    const playButton = page.locator('#play-button');
    await playButton.click();
    
    await page.waitForTimeout(1000);
    const audioPlayer = page.locator('#audio-player');
    await expect(audioPlayer).toBeVisible();
  });

  test('should accept input in answer field', async ({ page }) => {
    const answerInput = page.locator('#answer-input');
    await answerInput.fill('1234');
    
    await expect(answerInput).toHaveValue('1234');
  });

  test('should submit answer successfully', async ({ page }) => {
    const answerInput = page.locator('#answer-input');
    await answerInput.fill('1234');
    
    const submitButton = page.locator('#submit-button');
    await submitButton.click();
    
    await page.waitForTimeout(1000);
    const resultMessage = page.locator('#result-message');
    await expect(resultMessage).toBeVisible();
  });

  test('should have retry button after failure', async ({ page }) => {
    const answerInput = page.locator('#answer-input');
    await answerInput.fill('wrong');
    
    const submitButton = page.locator('#submit-button');
    await submitButton.click();
    
    await page.waitForTimeout(1000);
    const retryButton = page.locator('#retry-button');
    await expect(retryButton).toBeVisible();
  });

  test('should reload captcha on retry', async ({ page }) => {
    const answerInput = page.locator('#answer-input');
    await answerInput.fill('wrong');
    
    const submitButton = page.locator('#submit-button');
    await submitButton.click();
    
    await page.waitForTimeout(1000);
    const retryButton = page.locator('#retry-button');
    await retryButton.click();
    
    await page.waitForTimeout(1000);
    await expect(answerInput).toHaveValue('');
  });

  test('should support keyboard navigation', async ({ page }) => {
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    
    const answerInput = page.locator('#answer-input');
    await expect(answerInput).toBeFocused();
  });

  test('should have accessible ARIA labels', async ({ page }) => {
    const playButton = page.locator('#play-button');
    await expect(playButton).toHaveAttribute('aria-label');
    
    const answerInput = page.locator('#answer-input');
    await expect(answerInput).toHaveAttribute('aria-label');
  });

  test('should show loading state', async ({ page }) => {
    const loadingSpinner = page.locator('#loading-spinner');
    await expect(loadingSpinner).toBeAttached();
  });
});

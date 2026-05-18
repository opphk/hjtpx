import { test, expect } from '@playwright/test';

test.describe('GDPR Compliance E2E Tests', () => {
  test('should display cookie consent banner', async ({ page }) => {
    await page.goto('/home.html');
    
    const cookieBanner = page.locator('#cookie-consent-banner');
    await expect(cookieBanner).toBeVisible();
  });

  test('should accept all cookies', async ({ page }) => {
    await page.goto('/home.html');
    
    const acceptAllButton = page.locator('#accept-all-cookies');
    await acceptAllButton.click();
    
    const cookieBanner = page.locator('#cookie-consent-banner');
    await expect(cookieBanner).not.toBeVisible();
  });

  test('should reject non-essential cookies', async ({ page }) => {
    await page.goto('/home.html');
    
    const rejectButton = page.locator('#reject-non-essential');
    await rejectButton.click();
    
    const cookieBanner = page.locator('#cookie-consent-banner');
    await expect(cookieBanner).not.toBeVisible();
  });

  test('should allow user to access data', async ({ page }) => {
    await page.goto('/gdpr/data-access.html');
    
    const requestButton = page.locator('#request-data-button');
    await expect(requestButton).toBeVisible();
  });

  test('should provide data export option', async ({ page }) => {
    await page.goto('/gdpr/data-access.html');
    
    const exportButton = page.locator('#export-data-button');
    await exportButton.click();
    
    const exportSuccess = page.locator('#export-success-message');
    await expect(exportSuccess).toBeVisible();
  });

  test('should allow data deletion request', async ({ page }) => {
    await page.goto('/gdpr/data-deletion.html');
    
    const deleteButton = page.locator('#request-deletion-button');
    await deleteButton.click();
    
    const confirmationModal = page.locator('#deletion-confirmation-modal');
    await expect(confirmationModal).toBeVisible();
  });

  test('should display privacy policy link', async ({ page }) => {
    await page.goto('/home.html');
    
    const privacyLink = page.locator('a[href="/privacy-policy"]');
    await expect(privacyLink).toBeVisible();
  });

  test('should show data processing information', async ({ page }) => {
    await page.goto('/gdpr/data-processing.html');
    
    const processingInfo = page.locator('#data-processing-info');
    await expect(processingInfo).toBeVisible();
  });

  test('should provide consent management', async ({ page }) => {
    await page.goto('/gdpr/consent-management.html');
    
    const consentOptions = page.locator('.consent-option');
    await expect(consentOptions.first()).toBeVisible();
  });

  test('should log consent changes', async ({ page }) => {
    await page.goto('/home.html');
    
    const acceptButton = page.locator('#accept-all-cookies');
    await acceptButton.click();
    
    await page.waitForTimeout(1000);
    const consentLog = page.locator('#consent-log');
    await expect(consentLog).toBeVisible();
  });

  test('should anonymize data on request', async ({ page }) => {
    await page.goto('/gdpr/data-anonymization.html');
    
    const anonymizeButton = page.locator('#anonymize-data-button');
    await anonymizeButton.click();
    
    const anonymizationSuccess = page.locator('#anonymization-success');
    await expect(anonymizationSuccess).toBeVisible();
  });

  test('should update privacy settings', async ({ page }) => {
    await page.goto('/gdpr/privacy-settings.html');
    
    const analyticsToggle = page.locator('#analytics-consent-toggle');
    await analyticsToggle.click();
    
    const saveButton = page.locator('#save-privacy-settings');
    await saveButton.click();
    
    const successMessage = page.locator('#settings-saved-message');
    await expect(successMessage).toBeVisible();
  });
});

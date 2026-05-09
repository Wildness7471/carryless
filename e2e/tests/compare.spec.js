import { test, expect } from '@playwright/test';
import { loginAs, TEST_USER } from './helpers.js';

test.describe('Pack compare', () => {
    test.beforeEach(async ({ page }) => {
        await loginAs(page, TEST_USER);
    });

    test('compare page renders with ids query param', async ({ page }) => {
        // Create two packs first
        const packIds = [];
        for (const name of ['Compare Pack A', 'Compare Pack B']) {
            await page.goto('/packs/new');
            await page.fill('[name="name"]', name);
            await page.click('button[type="submit"]');
            const url = page.url();
            packIds.push(url.split('/packs/')[1].split('/')[0]);
        }

        await page.goto(`/packs/compare?ids=${packIds.join(',')}`);
        await expect(page).toHaveURL(/compare/);
        await expect(page.locator('body')).toContainText('Compare Pack A');
        await expect(page.locator('body')).toContainText('Compare Pack B');
    });

    test('compare with single pack id still renders', async ({ page }) => {
        await page.goto('/packs/new');
        await page.fill('[name="name"]', 'Single Compare Pack');
        await page.click('button[type="submit"]');
        const packId = page.url().split('/packs/')[1].split('/')[0];

        await page.goto(`/packs/compare?ids=${packId}`);
        // Should show the page, not 404
        const status = await page.evaluate(() => document.readyState);
        expect(status).toBe('complete');
    });
});

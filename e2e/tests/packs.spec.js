import { test, expect } from '@playwright/test';
import { loginAs, TEST_USER } from './helpers.js';

test.describe('Packs', () => {
    test.beforeEach(async ({ page }) => {
        await loginAs(page, TEST_USER);
    });

    test('packs list page renders', async ({ page }) => {
        await page.goto('/packs');
        await expect(page).toHaveURL('/packs');
        // Page should have the packs table or empty state
        await expect(page.locator('body')).toBeVisible();
    });

    test('create a new pack', async ({ page }) => {
        await page.goto('/packs/new');
        await expect(page.locator('[name="name"]')).toBeVisible();
        await page.fill('[name="name"]', 'E2E Test Pack');
        await page.click('button[type="submit"]');
        // Should redirect to pack detail
        await expect(page).toHaveURL(/\/packs\//);
    });

    test('pack detail shows pack name', async ({ page }) => {
        // Create pack
        await page.goto('/packs/new');
        await page.fill('[name="name"]', 'Detail Test Pack');
        await page.click('button[type="submit"]');
        await expect(page).toHaveURL(/\/packs\//);
        // Pack name should appear on the page
        await expect(page.locator('body')).toContainText('Detail Test Pack');
    });

    test('packs list shows created pack', async ({ page }) => {
        // Create a uniquely named pack
        const packName = 'Listing Test Pack ' + Date.now();
        await page.goto('/packs/new');
        await page.fill('[name="name"]', packName);
        await page.click('button[type="submit"]');

        // Go to packs list
        await page.goto('/packs');
        await expect(page.locator('body')).toContainText(packName);
    });

    test('delete a pack', async ({ page }) => {
        const packName = 'Delete Me Pack ' + Date.now();
        await page.goto('/packs/new');
        await page.fill('[name="name"]', packName);
        await page.click('button[type="submit"]');

        // Get pack ID from URL
        const url = page.url();
        const packId = url.split('/packs/')[1].split('/')[0];

        // Delete via form POST
        const csrfToken = await page.evaluate(() => {
            const meta = document.querySelector('meta[name="csrf-token"]');
            return meta ? meta.content : '';
        });

        await page.request.post(`/packs/${packId}/delete`, {
            form: { csrf_token: csrfToken },
        });

        await page.goto('/packs');
        await expect(page.locator('body')).not.toContainText(packName);
    });
});

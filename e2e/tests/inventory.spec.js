import { test, expect } from '@playwright/test';
import { loginAs, TEST_USER } from './helpers.js';

test.describe('Inventory', () => {
    test.beforeEach(async ({ page }) => {
        await loginAs(page, TEST_USER);
    });

    test('inventory page renders', async ({ page }) => {
        await page.goto('/inventory');
        await expect(page).toHaveURL('/inventory');
        await expect(page.locator('body')).toBeVisible();
    });

    test('new item page renders', async ({ page }) => {
        await page.goto('/inventory/items/new');
        await expect(page.locator('[name="name"]')).toBeVisible();
        await expect(page.locator('[name="weight"]')).toBeVisible();
    });

    test('create a new item redirects back to inventory', async ({ page }) => {
        // First ensure there's a category — navigate to categories and create one
        await page.goto('/categories/new');
        const catNameField = page.locator('[name="name"]');
        if (await catNameField.isVisible()) {
            await catNameField.fill('E2E Category');
            await page.click('button[type="submit"]');
        }

        await page.goto('/inventory/items/new');
        await page.fill('[name="name"]', 'E2E Test Item');
        await page.fill('[name="weight"]', '100');

        // Select the first category if dropdown exists
        const categorySelect = page.locator('[name="category_id"]');
        if (await categorySelect.isVisible()) {
            await categorySelect.selectOption({ index: 1 });
        }

        await page.click('button[type="submit"]');
        await expect(page).toHaveURL(/\/(inventory|items)/);
    });

    test('CSV export returns a file', async ({ page }) => {
        const [download] = await Promise.all([
            page.waitForEvent('download'),
            page.goto('/inventory/export'),
        ]);
        expect(download.suggestedFilename()).toMatch(/\.csv$/i);
    });
});

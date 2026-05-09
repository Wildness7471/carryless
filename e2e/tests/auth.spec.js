import { test, expect } from '@playwright/test';
import { TEST_USER, loginAs } from './helpers.js';

test.describe('Auth flows', () => {
    test('register page renders', async ({ page }) => {
        await page.goto('/register');
        await expect(page.locator('#username')).toBeVisible();
        await expect(page.locator('#email')).toBeVisible();
        await expect(page.locator('#password')).toBeVisible();
        await expect(page.locator('#confirm_password')).toBeVisible();
    });

    test('register with short username shows error', async ({ page }) => {
        await page.goto('/register');
        await page.fill('#username', 'ab');
        await page.fill('#email', 'newuser@example.com');
        await page.fill('#password', 'password123');
        await page.fill('#confirm_password', 'password123');
        await page.click('button[type="submit"]');
        // Should stay on register page with error
        await expect(page).toHaveURL(/register/);
    });

    test('register with mismatched passwords shows error', async ({ page }) => {
        await page.goto('/register');
        await page.fill('#username', 'newuser');
        await page.fill('#email', 'newuser2@example.com');
        await page.fill('#password', 'password123');
        await page.fill('#confirm_password', 'different999');
        await page.click('button[type="submit"]');
        await expect(page).toHaveURL(/register/);
    });

    test('login page renders', async ({ page }) => {
        await page.goto('/login');
        await expect(page.locator('#email')).toBeVisible();
        await expect(page.locator('#password')).toBeVisible();
    });

    test('login with valid credentials redirects', async ({ page }) => {
        await loginAs(page, TEST_USER);
        await expect(page).toHaveURL(/\/(dashboard|packs)/);
    });

    test('login with wrong password stays on login', async ({ page }) => {
        await page.goto('/login');
        await page.fill('#email', TEST_USER.email);
        await page.fill('#password', 'wrongpassword');
        await page.click('button[type="submit"]');
        await expect(page).toHaveURL(/login/);
    });

    test('logout redirects to home or login', async ({ page }) => {
        await loginAs(page, TEST_USER);
        // Find and submit the logout form/button
        await page.evaluate(async (csrf) => {
            const resp = await fetch('/logout', {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: 'csrf_token=' + encodeURIComponent(csrf),
                redirect: 'manual',
            });
        }, '');
        await page.goto('/packs');
        await expect(page).toHaveURL(/\/(login|register|$)/);
    });

    test('authenticated user redirected away from login', async ({ page }) => {
        await loginAs(page, TEST_USER);
        await page.goto('/login');
        // Should be redirected since already logged in
        await expect(page).not.toHaveURL(/login/);
    });
});

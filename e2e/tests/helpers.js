// Shared helper for E2E tests.
// Assumes the server is running at BASE_URL and the seed user exists.

export const TEST_USER = {
    username: 'e2etester',
    email: 'e2e@example.com',
    password: 'password123',
};

export const TEST_ADMIN = {
    username: 'e2eadmin',
    email: 'admin@example.com',
    password: 'adminpass123',
};

/** Log in with the test user and wait for redirect to dashboard. */
export async function loginAs(page, user = TEST_USER) {
    await page.goto('/login');
    await page.fill('#email', user.email);
    await page.fill('#password', user.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|packs)/);
}

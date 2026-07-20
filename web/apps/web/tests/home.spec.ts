import { test, expect } from '@playwright/test';

test('homepage has title and loads correctly', async ({ page }) => {
  await page.goto('/');

  // Check that the page has a title
  const title = await page.title();
  expect(title).toBeTruthy();

  // Check that the page loads without errors
  const body = await page.locator('body');
  await expect(body).toBeVisible();
});

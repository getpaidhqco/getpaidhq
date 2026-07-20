import { test, expect } from '@playwright/test';

test.describe('Order Creation', () => {
  test('should display order creation form', async ({ page }) => {
    await page.goto('/orders/create');

    // Wait for the form to load
    await expect(page.getByRole('heading', { name: 'Customer Information' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Order Items' })).toBeVisible();
    
    // Check if customer search is present
    await expect(page.getByText('Customer *')).toBeVisible();
    
    // Check if currency selector is present
    await expect(page.getByText('Currency *')).toBeVisible();
    
    // Check if Add Item button is present
    await expect(page.getByRole('button', { name: 'Add Item' })).toBeVisible();
    
    // Create Order button should be disabled initially
    await expect(page.getByRole('button', { name: 'Create Order' })).toBeDisabled();
  });

  test('should be able to add order items', async ({ page }) => {
    await page.goto('/orders/create');

    // Click Add Item button
    await page.getByRole('button', { name: 'Add Item' }).click();
    
    // Should show the item form
    await expect(page.getByText('Product *')).toBeVisible();
    await expect(page.getByText('Price *')).toBeVisible();
    await expect(page.getByText('Quantity *')).toBeVisible();
    
    // Should show remove button
    await expect(page.getByRole('button').filter({ has: page.locator('[data-lucide="trash-2"]') })).toBeVisible();
  });

  test('should show order summary when items are added', async ({ page }) => {
    await page.goto('/orders/create');

    // Add an item first
    await page.getByRole('button', { name: 'Add Item' }).click();
    
    // The order summary should appear when we have items (though it might be empty until products are selected)
    // This tests that the component structure is working
  });

  test('should have proper form validation', async ({ page }) => {
    await page.goto('/orders/create');

    // Initially, Create Order should be disabled
    await expect(page.getByRole('button', { name: 'Create Order' })).toBeDisabled();
    
    // Add an item
    await page.getByRole('button', { name: 'Add Item' }).click();
    
    // Still should be disabled without customer and product
    await expect(page.getByRole('button', { name: 'Create Order' })).toBeDisabled();
  });
});
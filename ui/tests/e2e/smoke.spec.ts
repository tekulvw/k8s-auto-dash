import { test, expect } from '@playwright/test';

test('renders discovered tiles in groups', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByText('Jellyfin', { exact: true })).toBeVisible();
  await expect(page.getByText('Grafana', { exact: true })).toBeVisible();
  await expect(page.getByText('Media', { exact: true })).toBeVisible();
  await expect(page.getByText('Infrastructure', { exact: true })).toBeVisible();
});

test('search filters tiles', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Search tiles').fill('jelly');
  await expect(page.getByText('Jellyfin', { exact: true })).toBeVisible();
  await expect(page.getByText('Grafana')).not.toBeVisible();
});

test('edit mode shows action buttons', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Toggle edit mode').click();
  await expect(page.getByLabel('Edit tile').first()).toBeVisible();
});

test('opening tile editor shows k8s info', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Toggle edit mode').click();
  await page.getByLabel('Edit tile').first().click();
  await expect(page.getByText('Namespace')).toBeVisible();
  await expect(page.getByText('media', { exact: true })).toBeVisible();
});

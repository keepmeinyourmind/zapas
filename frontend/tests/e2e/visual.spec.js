import { test, expect } from '@playwright/test';
import { readFile, mkdir } from 'node:fs/promises';
import { mockApi } from './fixtures';

test('preserves the original savings card geometry and captures the React dashboard', async ({ page }) => {
  await mkdir('artifacts', { recursive: true });
  const original = await readFile('../mockups/zapas-dashboard.html', 'utf8');
  await page.route('**/__reference', (route) => route.fulfill({
    contentType: 'text/html',
    body: original.replace(/https:\/\/cdn.jsdelivr.net\/npm\/bootstrap@[^" ]+\/dist\/css\/bootstrap.min.css/, '/node_modules/bootstrap/dist/css/bootstrap.min.css').replace(/ integrity="[^"]*"/g, ''),
  }));
  await page.goto('/__reference');
  await page.evaluate(() => document.fonts.ready);
  const baseline = await page.locator('.savings-overview').boundingBox();
  await page.locator('.savings-overview').screenshot({ path: 'artifacts/original-savings-card.png' });
  await mockApi(page);
  await page.goto('/');
  await expect(page.locator('.savings-name')).toHaveText('Финансовая подушка');
  await page.evaluate(() => document.fonts.ready);
  const current = await page.locator('.savings-overview').boundingBox();
  expect(current.width).toBe(baseline.width);
  expect(Math.abs(current.height - baseline.height)).toBeLessThanOrEqual(1);
  expect(current.x).toBe(baseline.x);
  expect(current.y).toBe(baseline.y);
  await page.locator('.savings-overview').screenshot({ path: 'artifacts/react-savings-card.png' });
  await page.screenshot({ path: 'artifacts/dashboard-desktop.png', fullPage: true });
});

test('captures the mobile dashboard and rule modal', async ({ page }) => {
  await mkdir('artifacts', { recursive: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await mockApi(page);
  await page.goto('/');
  await expect(page.locator('.savings-name')).toHaveText('Финансовая подушка');
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({ path: 'artifacts/dashboard-mobile.png', fullPage: true });
  await page.getByRole('button', { name: 'Правило', exact: true }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await expect(page.locator('.modal.show')).toHaveCSS('opacity', '1');
  await page.screenshot({ path: 'artifacts/rule-mobile.png', fullPage: true, animations: 'disabled' });
});

import { test, expect } from '@playwright/test';
import { mockApi } from './fixtures';

test('routes decision notifications to decisions and removes inline dashboard questions', async ({ page }) => {
  await mockApi(page);
  await page.goto('/');
  await expect(page.getByRole('button', { name: 'Отложить 1 000 ₽', exact: true })).toHaveCount(0);
  const notification = page.getByRole('status', { name: 'Уведомление о решении' });
  await expect(notification).toBeVisible();
  await notification.getByRole('link', { name: 'Перейти к решению' }).click();
  await expect(page).toHaveURL('/decisions');
  await expect(notification).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Отложить 1 000 ₽', exact: true })).toBeVisible();
});

test('asks about payments before showing the amount and sends zero when unchecked', async ({ page }) => {
  const state = await mockApi(page, { missing: true });
  await page.goto('/onboarding');
  await expect(page.getByLabel('Обязательные платежи в месяц, ₽')).toHaveCount(0);
  await page.getByLabel('Есть обязательные платежи', { exact: true }).check();
  await page.getByLabel('Обязательные платежи в месяц, ₽').fill('7000');
  await page.getByLabel('Есть обязательные платежи', { exact: true }).uncheck();
  await page.getByLabel('Дата рождения').fill('2000-01-01');
  await page.getByLabel('Доход в месяц, ₽', { exact: true }).fill('30000');
  await page.getByLabel('Сколько месяцев есть доход').fill('8');
  await page.getByLabel('Согласие на обработку персональных данных').check();
  await page.getByRole('button', { name: 'Проверить анкету', exact: true }).click();
  await expect(page.getByText('Шаг 2 из 2 · Цель и правило', { exact: true })).toBeVisible();
  expect(state.calls.find((call) => call.path.endsWith('/onboarding')).body.monthly_payments).toBe(0);
});

test('can disable quiet hours and select a timezone', async ({ page }) => {
  const state = await mockApi(page);
  await page.goto('/');
  await page.getByRole('button', { name: 'Редактировать', exact: true }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByLabel('Откладывать, %').fill('10');
  await dialog.getByLabel('Срок', { exact: true }).selectOption('12');
  await expect(dialog.getByLabel('Не беспокоить в тихие часы')).not.toBeChecked();
  await expect(dialog.getByLabel('Тихие часы с')).toHaveCount(0);
  await dialog.getByText('Дополнительные параметры', { exact: true }).click();
  await dialog.getByLabel('Часовой пояс', { exact: false }).selectOption('Asia/Vladivostok');
  await dialog.getByRole('button', { name: 'Сохранить', exact: true }).click();
  await expect(dialog).not.toBeVisible();
  const rule = state.calls.find((call) => call.method === 'PUT');
  expect(rule.body.quiet_from).toBe('00:00');
  expect(rule.body.quiet_to).toBe('00:00');
  expect(rule.body.timezone).toBe('Asia/Vladivostok');
});

test('information is read-only for editing and header links replace footer links', async ({ page }) => {
  await mockApi(page);
  await page.goto('/savings');
  await expect(page.getByRole('heading', { name: 'Информация', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Редактировать', exact: true })).toHaveCount(0);
  await expect(page.getByRole('link', { name: 'Обновить анкету', exact: true })).toHaveCount(0);
  await expect(page.locator('footer')).toHaveCount(0);
  await page.getByRole('navigation', { name: 'Навигация в шапке' }).getByRole('link', { name: 'Личный кабинет' }).click();
  await expect(page).toHaveURL('/');
});

test('consent document link is separate from the checkbox and loans block is removed', async ({ page }) => {
  await mockApi(page, { missing: true });
  await page.goto('/onboarding');
  await expect(page.getByLabel('Есть кредиты, рассрочки или займы')).toHaveCount(0);
  await page.getByRole('button', { name: 'обработку персональных данных', exact: true }).click();
  await expect(page.getByRole('dialog')).toContainText('Документ будет добавлен позже.');
  await page.getByRole('button', { name: 'Закрыть', exact: true }).click();
  await expect(page.getByLabel('Согласие на обработку персональных данных')).not.toBeChecked();
});

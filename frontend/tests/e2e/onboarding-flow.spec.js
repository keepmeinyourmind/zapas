import { test, expect } from '@playwright/test';
import { mockApi } from './fixtures';

async function approveQuestionnaire(page) {
  await page.getByLabel('Дата рождения').fill('2006-03-15');
  await page.getByLabel('Доход в месяц, ₽', { exact: true }).fill('30000');
  await page.getByLabel('Сколько месяцев есть доход').fill('8');
  await page.getByLabel('Обязательные платежи в месяц, ₽').fill('7000');
  await page.getByLabel('Согласие на обработку персональных данных').check();
  await page.getByRole('button', { name: 'Проверить анкету', exact: true }).click();
}

test('a new client sees questionnaire first on every entry point', async ({ page }) => {
  const state = await mockApi(page, { missing: true });
  for (const path of ['/', '/savings', '/decisions', '/transactions', '/recurring']) {
    await page.goto(path);
    await expect(page.getByRole('link', { name: 'Заполнить анкету', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: /Настроить правило|Изменить правило/ })).toHaveCount(0);
    await expect(page.getByRole('alert')).toHaveCount(0);
  }
  expect(state.calls.some((call) => /\/(decisions|transactions|recurring)(\?|$)/.test(call.path))).toBe(false);
  expect(state.calls.some((call) => call.method !== 'GET')).toBe(false);
});

test('connects in order: questionnaire, rule, connected dashboard', async ({ page }) => {
  const state = await mockApi(page, { missing: true });
  await page.goto('/');
  await page.getByRole('link', { name: 'Заполнить анкету', exact: true }).click();
  await expect(page.getByText('Шаг 1 из 2 · Анкета', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Проверить и подключить' })).toHaveCount(0);
  await approveQuestionnaire(page);
  await expect(page.getByText('Шаг 2 из 2 · Цель и правило', { exact: true })).toBeVisible();
  await expect(page.getByLabel('Дата рождения')).toHaveCount(0);
  await page.getByRole('button', { name: 'Настроить правило', exact: true }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByLabel('Название цели').fill('Подушка');
  await dialog.getByLabel('Сумма цели, ₽', { exact: true }).fill('36000');
  await dialog.getByLabel('Откладывать, %').fill('10');
  await dialog.getByLabel('Срок', { exact: true }).selectOption('12');
  await dialog.getByRole('button', { name: 'Подключить копилку', exact: true }).click();
  await expect(page).toHaveURL('/');
  await expect(page.getByRole('dialog')).not.toBeVisible();
  await expect(page.locator('.savings-name')).toHaveText('Подушка');
  const mutations = state.calls.filter((call) => call.method !== 'GET');
  expect(mutations.map((call) => call.path)).toEqual(['/api/clients/anya/onboarding', '/api/clients/anya/rule']);
});

test('resumes step two across routes, modal close and reload without storing questionnaire fields', async ({ page }) => {
  await mockApi(page, { missing: true });
  await page.goto('/onboarding');
  await approveQuestionnaire(page);
  await page.getByRole('button', { name: 'Настроить правило', exact: true }).click();
  await page.getByRole('button', { name: 'Закрыть', exact: true }).click();
  await page.getByRole('link', { name: 'Назад на главную' }).click();
  await expect(page.getByRole('button', { name: 'Настроить правило', exact: true })).toBeVisible();
  await page.reload();
  await page.getByRole('button', { name: 'Настроить правило', exact: true }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  const receipt = await page.evaluate(() => JSON.parse(sessionStorage.getItem('zapas:onboarding:anya')));
  expect(receipt).toEqual({ client_id: 'anya', eligible: true, mode: 'simple' });
  await page.getByRole('button', { name: 'Закрыть', exact: true }).click();
  await page.goto('/savings');
  await expect(page.getByRole('button', { name: 'Настроить правило', exact: true })).toBeVisible();
});

test('refusal leaves the client on step one and never opens rule creation', async ({ page }) => {
  const state = await mockApi(page, { missing: true, onboarding: { client_id: 'anya', eligible: false, mode: 'smart', score: 0, reasons: ['Копилка доступна с 18 лет'] } });
  await page.goto('/onboarding');
  await approveQuestionnaire(page);
  await expect(page.getByText('Копилка доступна с 18 лет', { exact: true })).toBeVisible();
  await expect(page.getByText('Шаг 1 из 2 · Анкета', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Настроить правило' })).toHaveCount(0);
  await page.goto('/savings');
  await expect(page.getByRole('link', { name: 'Заполнить анкету', exact: true })).toBeVisible();
  expect(state.calls.some((call) => call.method === 'PUT')).toBe(false);
});

test('questionnaire errors do not unlock rule setup', async ({ page }) => {
  await mockApi(page, { missing: true, override: async ({ path, json }) => {
    if (!path.endsWith('/onboarding')) return false;
    await json({ error: 'Не удалось сохранить анкету' }, 503);
    return true;
  } });
  await page.goto('/onboarding');
  await approveQuestionnaire(page);
  await expect(page.getByRole('alert')).toContainText('Не удалось сохранить анкету');
  await expect(page.getByRole('button', { name: 'Настроить правило' })).toHaveCount(0);
  await page.goto('/');
  await expect(page.getByRole('link', { name: 'Заполнить анкету', exact: true })).toBeVisible();
});

test('a repeated questionnaire refusal replaces the previous approval', async ({ page }) => {
  const state = await mockApi(page, { missing: true });
  await page.goto('/onboarding');
  await approveQuestionnaire(page);
  await page.getByRole('button', { name: 'Заполнить анкету заново', exact: true }).click();
  await expect(page.getByText('Шаг 1 из 2 · Анкета', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Настроить правило' })).toHaveCount(0);
  state.onboarding = { client_id: 'anya', eligible: false, mode: 'simple', score: 0, reasons: ['Недостаточно дохода для накоплений'] };
  await approveQuestionnaire(page);
  await expect(page.getByText('Недостаточно дохода для накоплений', { exact: true })).toBeVisible();
  await page.goto('/');
  await expect(page.getByRole('link', { name: 'Заполнить анкету', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Настроить правило' })).toHaveCount(0);
  const receipt = await page.evaluate(() => JSON.parse(sessionStorage.getItem('zapas:onboarding:anya')));
  expect(receipt.eligible).toBe(false);
});

test('a stale local approval cannot bypass server eligibility and returns to step one', async ({ page }) => {
  const state = await mockApi(page, { missing: true });
  await page.goto('/onboarding');
  await approveQuestionnaire(page);
  state.eligible = false;
  await page.getByRole('button', { name: 'Настроить правило', exact: true }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByLabel('Название цели').fill('Подушка');
  await dialog.getByLabel('Сумма цели, ₽', { exact: true }).fill('36000');
  await dialog.getByLabel('Откладывать, %').fill('10');
  await dialog.getByLabel('Срок', { exact: true }).selectOption('12');
  await dialog.getByRole('button', { name: 'Подключить копилку', exact: true }).click();
  await expect(page.getByRole('dialog')).not.toBeVisible();
  await expect(page.getByRole('button', { name: 'Проверить анкету', exact: true })).toBeVisible();
  expect(await page.evaluate(() => sessionStorage.getItem('zapas:onboarding:anya'))).toBeNull();
  await page.goto('/savings');
  await expect(page.getByRole('link', { name: 'Заполнить анкету', exact: true })).toBeVisible();
});

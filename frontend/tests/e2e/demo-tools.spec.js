import { test, expect } from '@playwright/test';
import { mockApi } from './fixtures';

test('checks health and recovers when the API becomes available', async ({ page }) => {
  let unavailable = true;
  const state = await mockApi(page, { override: async ({ path, json }) => {
    if (path === '/health' && unavailable) { await json({ error: 'Соединение недоступно' }, 503); return true; }
    return false;
  } });
  await page.goto('/demo');
  await expect(page.getByText('API недоступен', { exact: true })).toBeVisible();
  unavailable = false;
  await page.getByRole('button', { name: 'Проверить соединение', exact: true }).click();
  await expect(page.getByText('API работает', { exact: true })).toBeVisible();
  expect(state.calls.filter((call) => call.path === '/health').length).toBeGreaterThanOrEqual(2);
});

for (const type of ['incoming', 'outgoing', 'refund', 'internal', 'loan']) {
  test(`posts a ${type} transaction with kopecks, client ID and Moscow time`, async ({ page }) => {
    const state = await mockApi(page);
    const originalBalance = state.savings.card_balance;
    await page.goto('/demo');
    await page.getByLabel('Тип операции', { exact: true }).selectOption(type);
    await page.getByLabel('Сумма операции, ₽', { exact: true }).fill('1000.01');
    await page.getByLabel('Отправитель или получатель', { exact: true }).fill('Тестовый контрагент');
    await page.getByLabel('Время операции (МСК)', { exact: true }).fill('2026-10-20T10:00');
    await page.getByRole('button', { name: 'Добавить операцию', exact: true }).click();
    await expect(page.getByText('Операция принята', { exact: true })).toBeVisible();
    const call = state.calls.find((call) => call.path === '/api/transactions' && call.method === 'POST');
    expect(call.body).toMatchObject({ client_id: 'anya', amount: 100001, type, counterparty: 'Тестовый контрагент', counterparty_kind: 'legal', occurred_at: '2026-10-20T07:00:00.000Z' });
    expect(call.body.id).toBeTruthy();
    await expect(page.getByLabel('Сумма операции, ₽', { exact: true })).toHaveValue('');
    const delta = type === 'outgoing' ? -100001 : type === 'loan' ? 0 : 100001;
    expect(state.savings.card_balance).toBe(originalBalance + delta);
    await page.getByRole('link', { name: 'Назад на главную', exact: true }).click();
    await expect(page.locator('.money-history-row').filter({ hasText: 'Тестовый контрагент' })).toBeVisible();
  });
}

test('retries the same transaction after a lost response and page reload without crediting twice', async ({ page }) => {
  let count = 0;
  const state = await mockApi(page, { override: async ({ route, path, method, body, state, json }) => {
    if (path !== '/api/transactions' || method !== 'POST') return false;
    count++;
    if (count === 1) { state.savings.card_balance += body.amount; await route.abort(); }
    else await json({ income_kind: 'none', decision: null });
    return true;
  } });
  const originalBalance = state.savings.card_balance;
  await page.goto('/demo');
  await page.getByLabel('Сумма операции, ₽', { exact: true }).fill('1000.01');
  await page.getByRole('button', { name: 'Добавить операцию', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Повторить отправку операции', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Сдвинуть время', exact: true })).toBeDisabled();
  await expect(page.getByLabel('Сумма операции, ₽', { exact: true })).toBeDisabled();
  await page.reload();
  await expect(page.getByLabel('Сумма операции, ₽', { exact: true })).toHaveValue('1000.01');
  await page.getByRole('button', { name: 'Повторить отправку операции', exact: true }).click();
  await expect(page.getByText('Операция принята', { exact: true })).toBeVisible();
  const calls = state.calls.filter((call) => call.path === '/api/transactions');
  expect(calls).toHaveLength(2);
  expect(calls[1].body).toEqual(calls[0].body);
  expect(state.savings.card_balance).toBe(originalBalance + 100001);
});

test('a definitive validation error unlocks the form for a corrected new request', async ({ page }) => {
  let fail = true;
  const state = await mockApi(page, { override: async ({ path, method, json }) => {
    if (path === '/api/transactions' && method === 'POST' && fail) { await json({ error: 'Некорректная сумма операции' }, 400); return true; }
    return false;
  } });
  await page.goto('/demo');
  await page.getByLabel('Сумма операции, ₽', { exact: true }).fill('1000');
  await page.getByRole('button', { name: 'Добавить операцию', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Некорректная сумма операции');
  await expect(page.getByLabel('Сумма операции, ₽', { exact: true })).toBeEnabled();
  fail = false;
  await page.getByLabel('Сумма операции, ₽', { exact: true }).fill('1500');
  await page.getByRole('button', { name: 'Добавить операцию', exact: true }).click();
  await expect(page.getByText('Операция принята', { exact: true })).toBeVisible();
  const calls = state.calls.filter((call) => call.path === '/api/transactions');
  expect(calls[0].body.id).not.toBe(calls[1].body.id);
});

test('imposes and lifts arrest with a newer timestamp and refreshes the savings state', async ({ page }) => {
  const state = await mockApi(page);
  await page.goto('/demo');
  await page.getByLabel('Время события (МСК)', { exact: true }).fill('2026-10-20T10:00');
  await page.getByRole('button', { name: 'Отправить событие', exact: true }).click();
  await expect(page.getByText('Событие принято. По данным сервера: на счёте действует арест.', { exact: true })).toBeVisible();
  expect(state.savings.arrested).toBe(true);
  await expect(page.getByLabel('Время события (МСК)', { exact: true })).toHaveValue('2026-10-20T10:00:01');
  await page.getByLabel('Событие', { exact: true }).selectOption('arrest_lifted');
  await page.getByRole('button', { name: 'Отправить событие', exact: true }).click();
  await expect(page.getByText('Событие принято. По данным сервера: на счёте нет ареста.', { exact: true })).toBeVisible();
  expect(state.savings.arrested).toBe(false);
  const calls = state.calls.filter((call) => call.path === '/api/events/account');
  expect(calls.map((call) => call.body.type)).toEqual(['arrest_imposed', 'arrest_lifted']);
  expect(new Date(calls[1].body.occurred_at) > new Date(calls[0].body.occurred_at)).toBe(true);
  await page.getByRole('link', { name: 'Назад на главную', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Снять', exact: true })).toBeEnabled();
});

test('shows the returned account state when an older lifting event is ignored', async ({ page }) => {
  const state = await mockApi(page);
  state.savings.arrested = true;
  state.lastEventAt = '2026-10-21T07:00:00Z';
  await page.goto('/demo');
  await page.getByLabel('Событие', { exact: true }).selectOption('arrest_lifted');
  await page.getByLabel('Время события (МСК)', { exact: true }).fill('2026-10-20T10:00');
  await page.getByRole('button', { name: 'Отправить событие', exact: true }).click();
  await expect(page.getByText('Событие принято. По данным сервера: на счёте действует арест.', { exact: true })).toBeVisible();
  expect(state.savings.arrested).toBe(true);
});

test('reuses account event payload after a lost response and reload', async ({ page }) => {
  let count = 0;
  const state = await mockApi(page, { override: async ({ route, path, state, json }) => {
    if (path !== '/api/events/account') return false;
    count++;
    if (count === 1) { state.savings.arrested = true; await route.abort(); }
    else await json({ client_id: 'anya', arrested: true });
    return true;
  } });
  await page.goto('/demo');
  await page.getByRole('button', { name: 'Отправить событие', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Повторить отправку события', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Добавить операцию', exact: true })).toBeDisabled();
  await page.reload();
  await page.getByRole('button', { name: 'Повторить отправку события', exact: true }).click();
  await expect(page.getByText('Событие принято. По данным сервера: на счёте действует арест.', { exact: true })).toBeVisible();
  const calls = state.calls.filter((call) => call.path === '/api/events/account');
  expect(calls).toHaveLength(2);
  expect(calls[1].body).toEqual(calls[0].body);
});

test('scenario and time shift update event forms and require reset confirmation', async ({ page }) => {
  const state = await mockApi(page);
  await page.goto('/demo');
  await expect(page.getByRole('button', { name: 'Запустить сценарий', exact: true })).toBeDisabled();
  await page.getByLabel('Разрешаю сбросить тестовые данные Ани').check();
  await page.getByRole('button', { name: 'Запустить сценарий', exact: true }).click();
  await expect(page.getByLabel('Время операции (МСК)', { exact: true })).toHaveValue('2026-10-20T10:00');
  await expect(page.getByLabel('Разрешаю сбросить тестовые данные Ани')).not.toBeChecked();
  await page.getByRole('button', { name: 'Сдвинуть время', exact: true }).click();
  await expect(page.getByLabel('Время операции (МСК)', { exact: true })).toHaveValue('2026-10-21T11:00');
  await expect(page.getByLabel('Время события (МСК)', { exact: true })).toHaveValue('2026-10-21T11:00');
  expect(state.calls.find((call) => call.path === '/api/debug/advance-time').body).toEqual({ hours: 25 });
});

test('the verification panel fits a 320px screen', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await mockApi(page);
  await page.goto('/demo');
  await expect(page.getByRole('heading', { name: 'Проверка копилки', exact: true })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path: 'artifacts/verification-panel-mobile.png', fullPage: true });
});

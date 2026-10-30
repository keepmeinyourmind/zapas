import { test, expect } from '@playwright/test';
import { mockApi, decisionFixture } from './fixtures';

test('keeps the dashboard layout, aligns balances and opens calculation details', async ({ page }) => {
  const errors = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await mockApi(page);
  await page.goto('/');
  await expect(page.locator('.savings-name').first()).toHaveText('Финансовая подушка');
  await expect(page.locator('.savings-amount').first()).toContainText('12 180');
  await expect(page.getByRole('navigation').getByRole('link')).toHaveCount(6);
  await expect(page.locator('.savings-edit')).toHaveCount(0);
  const values = page.locator('.stat-value');
  const left = await values.nth(0).boundingBox();
  const right = await values.nth(1).boundingBox();
  expect(left.y).toBe(right.y);
  expect((await page.locator('.savings-overview .progress').boundingBox()).height).toBe(10);
  await page.getByRole('button', { name: 'Как рассчитана безопасная сумма', exact: true }).click();
  await expect(page.getByRole('dialog')).toContainText('Неприкосновенный остаток');
  await expect(page.getByRole('dialog')).toContainText('Рассрочка за ноутбук');
  await page.getByRole('button', { name: 'Закрыть', exact: true }).click();
  expect(errors).toEqual([]);
});

test('saves a complete rule with integer kopecks and uses the shared button', async ({ page }) => {
  const state = await mockApi(page);
  await page.goto('/');
  await page.getByRole('button', { name: 'Правило', exact: true }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog.getByRole('button', { name: 'Отмена' })).toHaveCount(0);
  await expect(dialog.getByLabel('Откладывать, %')).toHaveValue('');
  await dialog.getByLabel('Название цели').fill('Отпуск');
  await dialog.getByLabel('Сумма цели, ₽', { exact: true }).fill('50000.01');
  await dialog.getByLabel('Откладывать, %').fill('15');
  await dialog.getByLabel('Срок', { exact: true }).selectOption('12');
  await expect(dialog.getByRole('button', { name: 'Сохранить' })).toHaveClass(/reference-button/);
  await dialog.getByRole('button', { name: 'Сохранить' }).click();
  await expect(dialog).not.toBeVisible();
  const call = state.calls.find((c) => c.method === 'PUT' && c.path.endsWith('/rule'));
  expect(call.body.goal_amount).toBe(5000001);
  expect(call.body.save_percent).toBe(15);
  await expect(page.locator('.savings-name').first()).toHaveText('Отпуск');
  await page.reload();
  await page.getByRole('button', { name: 'Правило', exact: true }).click();
  await expect(page.getByRole('dialog').getByLabel('Откладывать, %')).toHaveValue('');
  expect(state.calls.some((c) => c.path.endsWith('/rule') && c.method === 'GET')).toBe(false);
});

test('withdraws kopecks, refreshes balances and prevents double submit', async ({ page }) => {
  const state = await mockApi(page);
  await page.goto('/');
  await page.getByRole('button', { name: 'Снять', exact: true }).click();
  await page.getByRole('dialog').getByLabel('Сумма, ₽', { exact: true }).fill('1000.01');
  await page.getByRole('button', { name: 'Подтвердить', exact: true }).dblclick();
  await expect(page.getByRole('dialog')).not.toBeVisible();
  const calls = state.calls.filter((c) => c.path.endsWith('/withdraw'));
  expect(calls).toHaveLength(1);
  expect(calls[0].body.amount).toBe(100001);
  expect(calls[0].body.id).toBeTruthy();
  await expect(page.locator('.savings-amount').first()).toContainText('11 179,99');
});

test('reuses withdrawal ID after a lost response, modal close, reload and changed balance', async ({ page }) => {
  let count = 0;
  const state = await mockApi(page, { override: async ({ route, path, body, state, json }) => {
    if (!path.endsWith('/withdraw')) return false;
    count++;
    if (count === 1) { state.savings.balance -= body.amount; await route.abort(); }
    else await json(state.savings);
    return true;
  } });
  await page.goto('/');
  await page.getByRole('button', { name: 'Снять', exact: true }).click();
  await page.getByRole('dialog').getByLabel('Сумма, ₽', { exact: true }).fill('12180');
  await page.getByRole('button', { name: 'Подтвердить', exact: true }).click();
  await expect(page.getByRole('dialog').getByRole('alert')).toContainText('Не удалось связаться');
  await page.getByRole('button', { name: 'Закрыть', exact: true }).click();
  await page.reload();
  await page.getByRole('button', { name: 'Снять', exact: true }).click();
  await expect(page.getByRole('dialog').getByLabel('Сумма, ₽', { exact: true })).toHaveValue('12180');
  await page.getByRole('button', { name: 'Подтвердить', exact: true }).click();
  await expect(page.getByRole('dialog')).not.toBeVisible();
  const calls = state.calls.filter((c) => c.path.endsWith('/withdraw'));
  expect(calls).toHaveLength(2);
  expect(calls[1].body).toEqual(calls[0].body);
});

test('answers pending decisions and does not offer queued decisions for answering', async ({ page }) => {
  const state = await mockApi(page);
  state.decisions.push({ ...structuredClone(decisionFixture), id: 'queued-1', status: 'queued' });
  await page.goto('/decisions');
  const queued = page.locator('.decision-row').filter({ has: page.getByText('В очереди', { exact: true }) });
  await expect(queued.getByRole('button', { name: /Отложить|Пропустить/ })).toHaveCount(0);
  await page.getByRole('button', { name: 'Отложить 1 000 ₽', exact: true }).click();
  await expect(page.locator('.decision-row').first()).toContainText('Выполнено');
  const answer = state.calls.find((c) => c.path.endsWith('/answer'));
  expect(answer.body).toEqual({ choice: 'amount', amount: 100000 });
  await page.getByRole('button', { name: 'Завершённые' }).click();
  await expect(page.locator('.decision-row')).toHaveCount(1);
});

test('income confirmation may turn the same pending question into an amount question', async ({ page }) => {
  const state = await mockApi(page);
  state.decisions[0].action = 'confirm_income';
  await page.goto('/decisions');
  await page.getByRole('button', { name: 'Да, это мой доход' }).click();
  await expect(page.getByRole('button', { name: 'Отложить 1 000 ₽', exact: true })).toBeVisible();
  expect(state.calls.find((c) => c.path.endsWith('/answer')).body).toEqual({ choice: 'yes' });
});

test('handles closed question conflict by refreshing the decision', async ({ page }) => {
  await mockApi(page, { override: async ({ path, state, json }) => {
    if (!path.endsWith('/answer')) return false;
    state.decisions[0].status = 'expired';
    await json({ error: 'Время на ответ вышло' }, 409);
    return true;
  } });
  await page.goto('/decisions');
  await page.getByRole('button', { name: 'Пропустить', exact: true }).click();
  await expect(page.getByText('Время ответа истекло', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Пропустить', exact: true })).toHaveCount(0);
});

test('changes recurring exclusion with the correct boolean direction', async ({ page }) => {
  const state = await mockApi(page);
  await page.goto('/recurring');
  await page.getByRole('switch').click();
  await expect(page.getByRole('switch')).not.toBeChecked();
  await expect(page.getByText('Необязательный', { exact: true })).toBeVisible();
  expect(state.calls.find((c) => c.path.endsWith('/exclusion')).body).toEqual({ counterparty: 'Рассрочка за ноутбук', excluded: true });
});

test('pauses via PATCH and displays confirmed server rule fields', async ({ page }) => {
  const state = await mockApi(page);
  await page.goto('/savings');
  await page.getByRole('button', { name: 'Поставить на паузу', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Возобновить копилку' })).toBeVisible();
  expect(state.calls.find((c) => c.method === 'PATCH').body).toEqual({ paused: true });
  await expect(page.locator('.savings-rule')).toContainText('10%');
});

async function fillOnboarding(page) {
  await page.getByLabel('Дата рождения').fill('2006-03-15');
  await page.getByLabel('Доход в месяц, ₽', { exact: true }).fill('30000.01');
  await page.getByLabel('Сколько месяцев есть доход').fill('8');
  await page.getByLabel('Обязательные платежи в месяц, ₽').fill('7000');
  await page.getByLabel('Согласие на обработку персональных данных').check();
}
test('submits onboarding in kopecks and offers rule setup after approval', async ({ page }) => {
  const state = await mockApi(page, { missing: true });
  await page.goto('/onboarding');
  await fillOnboarding(page);
  await page.getByRole('button', { name: 'Проверить анкету', exact: true }).click();
  await expect(page.getByText('Копилка вам доступна', { exact: true })).toBeVisible();
  const body = state.calls.find((c) => c.path.endsWith('/onboarding')).body;
  expect(body.monthly_income).toBe(3000001);
  expect(body.analysis_consent).toBe(false);
  expect(body.income_months).toBe(8);
  await page.getByRole('button', { name: 'Настроить правило', exact: true }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
});

test('renders eligible false as a refusal even when HTTP status is 200', async ({ page }) => {
  await mockApi(page, { onboarding: { client_id: 'anya', eligible: false, mode: 'smart', score: 0, reasons: ['Копилка доступна с 18 лет'] } });
  await page.goto('/onboarding');
  await fillOnboarding(page);
  await page.getByRole('button', { name: 'Обновить анкету', exact: true }).click();
  await expect(page.getByText('Копилка доступна с 18 лет', { exact: true })).toBeVisible();
  await expect(page.getByText('Копилка вам доступна', { exact: true })).toHaveCount(0);
});

test('filters transactions and supports browser back navigation', async ({ page }) => {
  await mockApi(page);
  await page.goto('/');
  await page.getByRole('link', { name: 'Перейти в историю' }).click();
  await expect(page.locator('.money-history-row')).toHaveCount(1);
  await page.getByRole('button', { name: 'Зачисления' }).click();
  await expect(page.locator('.money-history-row')).toHaveCount(2);
  await page.goBack();
  await expect(page.getByRole('navigation')).toBeVisible();
});

test('disables withdrawals and answers while account is arrested', async ({ page }) => {
  const state = await mockApi(page);
  state.savings.arrested = true;
  await page.goto('/');
  await expect(page.getByRole('button', { name: 'Снять', exact: true })).toBeDisabled();
  await expect(page.getByRole('button', { name: 'Отложить 1 000 ₽', exact: true })).toBeDisabled();
});

test('does not invent balances on network errors and recovers on retry', async ({ page }) => {
  let offline = true;
  await mockApi(page, { override: async ({ path, method, json }) => {
    if (path.endsWith('/savings') && method === 'GET' && offline) { await json({ error: 'Временно недоступно' }, 503); return true; }
    return false;
  } });
  await page.goto('/');
  await expect(page.getByRole('alert').first()).toContainText('Временно недоступно');
  await expect(page.locator('.savings-amount')).toHaveCount(0);
  offline = false;
  await page.getByRole('button', { name: 'Повторить', exact: true }).first().click();
  await expect(page.locator('.savings-amount')).toBeVisible();
});

test('shows empty states and no stale safety calculation in simple mode', async ({ page }) => {
  const state = await mockApi(page, { transactions: [], recurring: [], decisions: [] });
  state.savings.mode = 'simple';
  await page.goto('/');
  await expect(page.getByText('Операций пока нет.')).toBeVisible();
  await expect(page.getByText('Пока нет расчёта')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Как рассчитана безопасная сумма', exact: true })).toBeDisabled();
});

for (const width of [320, 390, 768]) {
  test(`fits all screens at ${width}px without horizontal scrolling`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await mockApi(page);
    for (const path of ['/', '/savings', '/transactions', '/recurring', '/decisions', '/onboarding']) {
      await page.goto(path);
      await expect(page.locator('h1, .savings-amount').first()).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), path).toBe(true);
    }
  });
}

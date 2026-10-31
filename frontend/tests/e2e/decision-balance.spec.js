import { test, expect } from '@playwright/test';
import { mockApi } from './fixtures';

test('shows the actual smaller transfer returned by the server', async ({ page }) => {
  const reason = 'Вы выбрали 1 000 ₽, но сейчас безопасно только 500 ₽ — откладываем 500 ₽';
  const state = await mockApi(page, {
    override: async ({ path, state, json }) => {
      if (!path.endsWith('/answer')) return false;
      Object.assign(state.decisions[0], { status: 'executed', amount: 50000, reason });
      state.savings.card_balance -= 50000;
      state.savings.balance += 50000;
      state.savings.pending_decision_id = null;
      await json(state.decisions[0]);
      return true;
    },
  });
  await page.goto('/');
  await page.getByRole('link', { name: 'Перейти к решению' }).click();
  await page.getByRole('button', { name: 'Отложить 1 000 ₽', exact: true }).click();
  await expect(page.getByText('Отложено 500 ₽', { exact: true })).toBeVisible();
  await expect(page.getByText(reason, { exact: true }).first()).toBeVisible();
  await expect(page.getByRole('button', { name: 'Отложить 1 000 ₽', exact: true })).toHaveCount(0);
  await expect(page.getByText('Отложено 500 ₽', { exact: true })).toBeVisible();
  expect(state.decisions[0].amount).toBe(50000);
});

test('closes an unaffordable offer and explains why no money was transferred', async ({ page }) => {
  const reason = 'Пока вы решали, на карте стало меньше денег — до следующего дохода они понадобятся, не откладываем';
  const state = await mockApi(page, {
    override: async ({ path, state, json }) => {
      if (!path.endsWith('/answer')) return false;
      state.decisions[0].status = 'skipped';
      state.decisions[0].reason = reason;
      state.savings.pending_decision_id = null;
      await json(state.decisions[0]);
      return true;
    },
  });
  state.savings.card_balance = 50000;
  const originalSavings = state.savings.balance;
  await page.goto('/');
  await page.getByRole('link', { name: 'Перейти к решению' }).click();
  await page.getByRole('button', { name: 'Отложить 1 000 ₽', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Нужен ваш ответ' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Отложить 1 000 ₽', exact: true })).toHaveCount(0);
  await expect(page.getByText('Пополнение пропущено', { exact: true })).toBeVisible();
  await expect(page.getByText('На карте стало меньше денег.', { exact: true }).first()).toBeVisible();
  await expect(page.getByRole('img', { name: 'Есть решения системы' })).toHaveCount(0);
  expect(state.savings.card_balance).toBe(50000);
  expect(state.savings.balance).toBe(originalSavings);
  await page.reload();
  await expect(page.getByText('Пополнение пропущено', { exact: true })).toBeVisible();
});

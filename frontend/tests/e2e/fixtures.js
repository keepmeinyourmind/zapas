export const savingsFixture = {
  client_id: 'anya', goal_name: 'Финансовая подушка', goal_amount: 3600000,
  balance: 1218000, card_balance: 2600000, progress_percent: 34, paused: false,
  mode: 'smart', arrested: false, pending_decision_id: 'ask-1',
};
export const ruleFixture = {
  client_id: 'anya', goal_name: 'Финансовая подушка', goal_amount: 3600000,
  term_months: 12, save_percent: 10, min_income: 300000, min_transfer: 10000,
  declared_income: 3000000, paused: false, quiet_from: '23:00', quiet_to: '09:00', timezone: 'Europe/Moscow',
};
export const decisionFixture = {
  id: 'ask-1', client_id: 'anya', source_tx_id: 'tx-1', action: 'ask',
  amount: 100000, options: [50000, 100000], reason: 'Доход ниже обычного на 20%.', status: 'pending',
  created_at: '2026-10-28T10:00:00Z', expires_at: '2026-10-29T10:00:00Z',
  explanation: { card_balance: 2600000, upcoming_total: 2500000, buffer: 0, safe_amount: 100000,
    until: '2026-11-19T10:00:00Z', upcoming: [{ title: 'Рассрочка за ноутбук', amount: 600000, at: '2026-10-25T10:00:00Z' }], summary: 'Можно безопасно отложить до 1 000 ₽' },
};
export const transactionsFixture = [
  { id: 'tx-1', client_id: 'anya', amount: 2400000, type: 'incoming', counterparty: 'Подработка', counterparty_kind: 'legal', income_kind: 'freelance', occurred_at: '2026-10-28T10:00:00Z' },
  { id: 'tx-2', client_id: 'anya', amount: 600000, type: 'outgoing', counterparty: 'Рассрочка за ноутбук', counterparty_kind: 'legal', income_kind: 'none', occurred_at: '2026-10-20T10:00:00Z' },
  { id: 'tx-3', client_id: 'anya', amount: 500000, type: 'incoming', counterparty: 'Стипендия', counterparty_kind: 'legal', income_kind: 'scholarship', occurred_at: '2026-10-18T10:00:00Z' },
];

export async function mockApi(page, options = {}) {
  const state = {
    savings: structuredClone(savingsFixture), rule: structuredClone(ruleFixture),
    decisions: [structuredClone(decisionFixture)], transactions: structuredClone(transactionsFixture),
    recurring: [{ counterparty: 'Рассрочка за ноутбук', amount: 600000, day_of_month: 25, last_at: '2026-09-25T10:00:00Z', excluded: false }],
    eligible: !options.missing, clientExists: !options.missing, calls: [],
    postedTransactions: new Map(), accountEvents: new Set(), ...options,
  };
  await page.route('**/health', async (route) => {
    const json = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    state.calls.push({ path: '/health', method: 'GET', body: null });
    if (state.override && await state.override({ route, path: '/health', method: 'GET', body: null, state, json })) return;
    await json({ status: 'ok' });
  });
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = new URL(req.url());
    const path = url.pathname;
    if (!path.startsWith('/api/')) return route.continue();
    const method = req.method();
    const body = req.postDataJSON();
    state.calls.push({ path, method, body, search: url.search });
    const json = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    if (state.override && await state.override({ route, path, method, body, state, json })) return;
    if (path.endsWith('/savings') && method === 'GET') return state.missing ? json({ error: 'Не найдено' }, 404) : json(state.savings);
    if (path.endsWith('/savings/withdraw')) {
      if (state.savings.arrested) return json({ error: 'Счёт арестован' }, 409);
      state.savings.balance -= body.amount;
      state.savings.card_balance += body.amount;
      return json(state.savings, 201);
    }
    if (path === '/api/transactions' && method === 'POST') {
      if (!state.clientExists) return json({ error: 'Клиент не найден' }, 404);
      const saved = state.postedTransactions.get(body.id);
      if (saved) return JSON.stringify(saved.body) === JSON.stringify(body) ? json(saved.result) : json({ error: 'Операция с таким id уже есть — с другими данными' }, 409);
      const incomeKind = body.type === 'incoming' && body.counterparty_kind === 'legal' ? 'salary' : 'none';
      if (body.type === 'outgoing') state.savings.card_balance -= body.amount;
      else if (body.type !== 'loan') state.savings.card_balance += body.amount;
      state.transactions.unshift({ ...body, income_kind: incomeKind });
      const decision = incomeKind === 'salary' && !state.savings.arrested && !state.savings.paused ? {
        ...structuredClone(decisionFixture), id: `decision-${body.id}`, source_tx_id: body.id,
        amount: Math.floor(body.amount * state.rule.save_percent / 100), created_at: body.occurred_at,
        reason: 'Тестовое поступление. Выберите сумму накопления.',
      } : null;
      if (decision) { state.decisions.unshift(decision); state.savings.pending_decision_id = decision.id; }
      const result = { income_kind: incomeKind, decision };
      state.postedTransactions.set(body.id, { body, result });
      return json(result, 201);
    }
    if (path.endsWith('/transactions')) return json(state.transactions);
    if (path === '/api/events/account') {
      if (!state.clientExists) return json({ error: 'Клиент не найден' }, 404);
      const duplicate = state.accountEvents.has(body.id);
      if (!duplicate && (!state.lastEventAt || new Date(body.occurred_at) > new Date(state.lastEventAt))) {
        state.savings.arrested = body.type === 'arrest_imposed';
        state.lastEventAt = body.occurred_at;
        if (state.savings.arrested) {
          state.decisions.forEach((decision) => { if (['pending', 'queued'].includes(decision.status)) decision.status = 'expired'; });
          state.savings.pending_decision_id = null;
        }
      }
      state.accountEvents.add(body.id);
      return json({ client_id: 'anya', arrested: state.savings.arrested }, duplicate ? 200 : 201);
    }
    if (path.startsWith('/api/scenarios/')) {
      state.missing = false;
      state.eligible = true;
      state.clientExists = true;
      state.savings = { ...structuredClone(savingsFixture), balance: 1500000, card_balance: 2600000 };
      state.decisions = [{ ...structuredClone(decisionFixture), created_at: '2026-10-20T07:00:00Z' }];
      state.now = '2026-10-20T07:00:00Z';
      state.lastEventAt = null;
      state.postedTransactions.clear();
      state.accountEvents.clear();
      return json({ client_id: 'anya', scenario: path.split('/')[3], description: 'Сценарий выполнен.', savings: state.savings, decision: state.decisions[0] });
    }
    if (path === '/api/debug/advance-time') {
      state.now = new Date(new Date(state.now || '2026-10-20T07:00:00Z').getTime() + body.hours * 3600000).toISOString();
      const expired = state.decisions.filter((d) => d.status === 'pending');
      expired.forEach((d) => { d.status = 'expired'; });
      state.savings.pending_decision_id = null;
      return json({ now: state.now, expired });
    }
    if (path.endsWith('/decisions')) {
      const status = url.searchParams.get('status');
      return json(status ? state.decisions.filter((d) => d.status === status) : state.decisions);
    }
    if (path.endsWith('/answer')) {
      const d = state.decisions.find((d) => path.includes(`/${d.id}/`));
      if (d.status !== 'pending') return json({ error: 'Вопрос уже закрыт' }, 409);
      if (body.choice === 'yes') { d.action = 'ask'; d.reason = 'Доход подтверждён. Выберите сумму.'; }
      else {
        d.status = body.choice === 'amount' ? 'executed' : 'skipped';
        if (body.choice === 'amount') { d.amount = body.amount; state.savings.balance += body.amount; }
        state.savings.pending_decision_id = null;
      }
      return json(d);
    }
    if (path.endsWith('/rule')) {
      if (method === 'GET') return json({ error: 'GET /rule отсутствует в контракте' }, 405);
      if (method === 'PUT' && !state.eligible) return json({ error: 'Сначала заполните анкету: без неё копилку подключить нельзя' }, 409);
      state.rule = method === 'PUT' ? { ...body, client_id: 'anya' } : { ...state.rule, ...body };
      Object.assign(state.savings, { goal_name: state.rule.goal_name, goal_amount: state.rule.goal_amount, paused: state.rule.paused });
      state.missing = false;
      return json(state.rule);
    }
    if (path.endsWith('/recurring/exclusion')) { state.recurring[0].excluded = body.excluded; return json(state.recurring); }
    if (path.endsWith('/recurring')) return json(state.recurring);
    if (path.endsWith('/onboarding')) {
      const result = state.onboarding || { client_id: 'anya', eligible: true, mode: body.analysis_consent ? 'smart' : 'simple', score: 80, reasons: [] };
      state.eligible = result.eligible;
      state.clientExists = true;
      if (!result.eligible && !state.missing) state.savings.paused = true;
      return json(result);
    }
    return json({ error: 'Неожиданный маршрут теста' }, 404);
  });
  return state;
}

import { useState } from 'react';
import { useIsMutating, useQueryClient } from '@tanstack/react-query';
import { useOutletContext } from 'react-router-dom';
import { config } from '../app/config';
import { useToast } from '../app/providers';
import { useClientMutation, clientKey, resourceKey } from '../shared/api/hooks';
import AppButton from '../shared/ui/AppButton';
import { Field } from '../shared/ui/FormFields';
import { InlineError } from '../shared/ui/States';
import { formatDate } from '../shared/utils/date';
import { clearOnboarding } from '../features/onboarding/status';
import { runScenario, advanceTime } from '../features/demo/api';
import ServiceHealth from '../features/demo/ServiceHealth';
import TransactionForm from '../features/demo/TransactionForm';
import AccountEventForm from '../features/demo/AccountEventForm';
import { readBankRequest } from '../features/demo/pendingRequest';

const scenarios = { stable: 'Обычный доход', income_drop: 'Снижение дохода', simple_mode: 'Простой режим', self_transfer: 'Перевод себе', new_client: 'Первый доход', income_rise: 'Рост дохода' };
const clockKey = `zapas:demo-clock:${config.apiBaseUrl}`;
function readClock() {
  try {
    const value = sessionStorage.getItem(clockKey);
    if (value && Number.isFinite(new Date(value).getTime())) return value;
  } catch { /* Хранилище браузера может быть недоступно. */ }
  return null;
}

export default function DemoPage() {
  const { savings } = useOutletContext();
  const [name, setName] = useState('income_drop');
  const [confirmed, setConfirmed] = useState(false);
  const [hours, setHours] = useState('25');
  const [clock, setClock] = useState(readClock);
  const [transactionPending, setTransactionPending] = useState(() => Boolean(readBankRequest('transaction')));
  const [accountPending, setAccountPending] = useState(() => Boolean(readBankRequest('account-event')));
  const cache = useQueryClient();
  const toast = useToast();
  const busy = useIsMutating({ mutationKey: ['demo'] }) > 0;
  const uncertain = transactionPending || accountPending;
  function recordClock(value) {
    setClock(value);
    try { sessionStorage.setItem(clockKey, value); } catch { /* Значение остаётся в состоянии React. */ }
  }
  const run = useClientMutation(runScenario, {
    mutationKey: ['demo', 'scenario'],
    onSuccess: async (result) => {
      // У сценария self_transfer нет времени решения; дата его события зафиксирована в API.
      recordClock(result.decision?.created_at || '2026-10-20T07:00:00Z');
      clearOnboarding(cache);
      await cache.resetQueries({ queryKey: resourceKey('rule'), exact: true });
      cache.setQueryData(resourceKey('savings'), result.savings);
      setConfirmed(false);
      toast(result.description || 'Сценарий выполнен.');
    },
  });
  const advance = useClientMutation(advanceTime, {
    mutationKey: ['demo', 'clock'],
    onSuccess: (result) => { recordClock(result.now); toast('Время сдвинуто, состояние вопросов обновлено.'); },
  });
  return <section className="view" id="view-demo">
    <div className="mb-4"><h1 className="page-title">Проверка копилки</h1><p className="page-description">Сценарии, операции и события счёта</p></div>
    <div className="row g-3">
      <div className="col-12"><ServiceHealth /></div>
      <div className="col-12"><article className="card"><div className="card-body p-4">
        <h2 className="section-title mb-3">Готовые сценарии</h2>
        <p className="small text-secondary">Запуск сценария сбрасывает данные клиента Ани. Сдвиг времени действует на весь сервер до его перезапуска.</p>
        {config.clientId !== 'anya' && <p className="form-error">Сценарии изменяют клиента anya. Текущий клиент: {config.clientId}.</p>}
        <Field label="Сценарий" value={name} onChange={(e) => setName(e.target.value)} disabled={busy || uncertain}>{Object.entries(scenarios).map(([value, title]) => <option value={value} key={value}>{title}</option>)}</Field>
        <div className="form-check mt-3"><input className="form-check-input" type="checkbox" id="confirm-reset" checked={confirmed} disabled={busy || uncertain} onChange={(e) => setConfirmed(e.target.checked)} /><label className="form-check-label small" htmlFor="confirm-reset">Разрешаю сбросить тестовые данные Ани</label></div>
        <AppButton className="mt-3" onClick={() => { if (!busy && !uncertain && confirmed) run.mutate(name); }} pending={run.isPending} disabled={busy || uncertain || !confirmed || config.clientId !== 'anya'}>Запустить сценарий</AppButton>
        <InlineError error={run.error} />
        <form className="mt-4" onSubmit={(e) => { e.preventDefault(); if (!busy && !uncertain) advance.mutate(Number(hours)); }}>
          <Field label="Сдвиг времени, часов" type="number" min="1" step="1" required value={hours} disabled={busy || uncertain} onChange={(e) => setHours(e.target.value)} />
          <AppButton className="mt-3" type="submit" pending={advance.isPending} disabled={busy || uncertain}>Сдвинуть время</AppButton>
        </form>
        <InlineError error={advance.error} />
        {clock && <p className="form-text mt-3 mb-0">Выбранное сценарное время: {formatDate(clock, true)} МСК. После перезапуска бэкенда проверьте время в формах.</p>}
        <p className="form-text mt-3 mb-0">После запуска сценария или сдвига часов время новых событий подставляется в формы. Его можно изменить вручную.</p>
        {uncertain && <p className="form-text mt-3 mb-0">Сначала получите подтверждение отправленной операции или события. До этого сброс данных и сдвиг времени недоступны.</p>}
        <AppButton className="mt-3" onClick={() => cache.invalidateQueries({ queryKey: clientKey, predicate: (q) => q.queryKey[2] !== 'rule' })} disabled={busy}>Обновить данные</AppButton>
      </div></article></div>
      <div className="col-12"><TransactionForm clock={clock} busy={busy} blocked={accountPending} onPendingChange={setTransactionPending} /></div>
      <div className="col-12"><AccountEventForm clock={clock} busy={busy} blocked={transactionPending} onPendingChange={setAccountPending} savings={savings.data} /></div>
    </div>
  </section>;
}

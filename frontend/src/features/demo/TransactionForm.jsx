import { useEffect, useRef, useState } from 'react';
import { createTransaction } from './api';
import { useBankSubmission } from './hooks';
import { moscowDateTime, eventTimestamp } from './time';
import { useToast } from '../../app/providers';
import { Field } from '../../shared/ui/FormFields';
import AppButton from '../../shared/ui/AppButton';
import { InlineError } from '../../shared/ui/States';
import { formatMoney, moneyInput, toKopecks } from '../../shared/utils/money';

const types = { incoming: 'Зачисление', outgoing: 'Списание', refund: 'Возврат', internal: 'Внутренний перевод', loan: 'Кредитные средства' };
const parties = { legal: 'Организация', person: 'Физическое лицо', self: 'Собственный счёт', bank: 'Банк' };
const incomeNames = { salary: 'Зарплата', advance: 'Аванс', bonus: 'Премия', scholarship: 'Стипендия', freelance: 'Фриланс', none: 'Не доход' };

export default function TransactionForm({ clock, busy, blocked, onPendingChange }) {
  const submission = useBankSubmission('transaction', createTransaction, onPendingChange);
  const { attempt, mutation, error, send } = submission;
  const toast = useToast();
  const lastClock = useRef(clock);
  const [form, setForm] = useState(() => ({
    type: attempt?.type || 'incoming', amount: moneyInput(attempt?.amount),
    counterparty: attempt?.counterparty || '', counterparty_kind: attempt?.counterparty_kind || 'legal',
    occurred_at: moscowDateTime(attempt?.occurred_at || clock || new Date()),
  }));
  useEffect(() => {
    if (clock !== lastClock.current) {
      lastClock.current = clock;
      if (clock && !attempt) setForm((previous) => ({ ...previous, occurred_at: moscowDateTime(clock) }));
    }
  }, [clock, attempt]);
  const bind = (name) => ({ value: form[name], onChange: (e) => setForm({ ...form, [name]: e.target.value }) });
  async function submit(e) {
    e.preventDefault();
    if (busy || blocked) return;
    const result = await send(() => {
      const amount = toKopecks(form.amount);
      if (amount <= 0) throw new Error('Сумма должна быть больше нуля.');
      return { ...form, amount, counterparty: form.counterparty.trim(), occurred_at: eventTimestamp(form.occurred_at) };
    });
    if (result) {
      setForm((previous) => ({ ...previous, amount: '' }));
      toast('Операция принята. Балансы и решения обновлены.');
    }
  }
  return <article className="card"><div className="card-body p-4"><h2 className="section-title mb-3">Тестовая операция по карте</h2>
    <p className="text-secondary small">Добавьте поступление или расход. Зачисление может вызвать пополнение копилки или вопрос — результат определяет бэкенд.</p>
    <form onSubmit={submit}><fieldset disabled={busy || blocked || Boolean(attempt)}><div className="row g-3">
      <Field label="Тип операции" {...bind('type')}>{Object.entries(types).map(([value, label]) => <option value={value} key={value}>{label}</option>)}</Field>
      <Field label="Сумма операции, ₽" type="number" min="0.01" step="0.01" required {...bind('amount')} />
      <Field label="Отправитель или получатель" maxLength={200} placeholder="Например, работодатель или магазин" {...bind('counterparty')} />
      <Field label="Вид отправителя или получателя" {...bind('counterparty_kind')}>{Object.entries(parties).map(([value, label]) => <option value={value} key={value}>{label}</option>)}</Field>
      <Field label="Время операции (МСК)" type="datetime-local" step="1" required {...bind('occurred_at')} />
    </div></fieldset>
    {attempt && !mutation.isPending && <p className="form-text mt-3">Подтверждение не получено. Повторите тот же запрос — идентификатор и данные сохранены, повторного зачисления не будет.</p>}
    <InlineError error={error} />
    <AppButton type="submit" className="mt-3" pending={mutation.isPending} disabled={busy || blocked}>{attempt && !mutation.isPending ? 'Повторить отправку операции' : 'Добавить операцию'}</AppButton>
    </form>
    {mutation.isSuccess && mutation.data && <div className="status-notice mt-3" role="status"><strong>Операция принята</strong><p className="mt-2 mb-0">Классификация: {incomeNames[mutation.data.income_kind] || mutation.data.income_kind}.</p>{mutation.data.decision ? <><p className="mt-2 mb-0">{mutation.data.decision.reason}</p><p className="mt-2 mb-0">Сумма решения: {formatMoney(mutation.data.decision.amount)}. Подробности — в разделе «Решения».</p></> : <p className="mt-2 mb-0">Решение о пополнении копилки не требуется.</p>}</div>}
  </div></article>;
}

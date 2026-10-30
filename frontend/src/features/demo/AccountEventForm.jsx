import { useEffect, useRef, useState } from 'react';
import { createAccountEvent } from './api';
import { useBankSubmission } from './hooks';
import { moscowDateTime, eventTimestamp } from './time';
import { useToast } from '../../app/providers';
import { Field } from '../../shared/ui/FormFields';
import AppButton from '../../shared/ui/AppButton';
import { InlineError } from '../../shared/ui/States';

export default function AccountEventForm({ clock, busy, blocked, onPendingChange, savings }) {
  const { attempt, mutation, error, send } = useBankSubmission('account-event', createAccountEvent, onPendingChange);
  const toast = useToast();
  const lastClock = useRef(clock);
  const [form, setForm] = useState(() => ({ type: attempt?.type || 'arrest_imposed', occurred_at: moscowDateTime(attempt?.occurred_at || clock || new Date()) }));
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
    const result = await send(() => ({ ...form, occurred_at: eventTimestamp(form.occurred_at) }));
    if (result) {
      toast(result.arrested ? 'На счёте действует арест.' : 'Арест счёта снят.');
      setForm((previous) => ({ ...previous, occurred_at: moscowDateTime(new Date(eventTimestamp(previous.occurred_at)).getTime() + 1000) }));
    }
  }
  return <article className="card"><div className="card-body p-4"><h2 className="section-title mb-3">Событие по счёту</h2>
    <p className="text-secondary small">Арест блокирует переводы в копилку и снятие, а открытые вопросы закрываются. Чтобы продолжить проверку, отправьте снятие ареста с более поздним временем.</p>
    <p className="form-text">Текущий статус: {savings ? savings.arrested ? 'действует арест' : 'без ареста' : 'пока неизвестен'}</p>
    <form onSubmit={submit}><fieldset disabled={busy || blocked || Boolean(attempt)}><div className="row g-3">
      <Field label="Событие" {...bind('type')}><option value="arrest_imposed">Наложить арест</option><option value="arrest_lifted">Снять арест</option></Field>
      <Field label="Время события (МСК)" type="datetime-local" step="1" required {...bind('occurred_at')} />
    </div></fieldset>
    {attempt && !mutation.isPending && <p className="form-text mt-3">Подтверждение не получено. Повтор использует прежний идентификатор и время события.</p>}
    <InlineError error={error} />
    <AppButton className="mt-3" type="submit" pending={mutation.isPending} disabled={busy || blocked}>{attempt && !mutation.isPending ? 'Повторить отправку события' : 'Отправить событие'}</AppButton>
    </form>
    {mutation.isSuccess && mutation.data && <p className="status-notice mt-3 mb-0" role="status">Событие принято. По данным сервера: {mutation.data.arrested ? 'на счёте действует арест.' : 'на счёте нет ареста.'}</p>}
  </div></article>;
}

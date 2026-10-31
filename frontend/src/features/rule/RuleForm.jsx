import { useState } from 'react';
import Modal from 'react-bootstrap/Modal';
import { useKnownRule, useSaveRule } from './hooks';
import { useToast } from '../../app/providers';
import AppButton from '../../shared/ui/AppButton';
import { Field, Checkbox } from '../../shared/ui/FormFields';
import { InlineError } from '../../shared/ui/States';
import { moneyInput, toKopecks } from '../../shared/utils/money';
import { useQueryClient } from '@tanstack/react-query';
import { clearOnboarding } from '../onboarding/status';

export default function RuleForm({ savings, onSaved, onRequiresOnboarding }) {
  const cache = useQueryClient();
  const { data: known } = useKnownRule();
  const mutation = useSaveRule();
  const toast = useToast();
  const [error, setError] = useState(null);
  const [form, setForm] = useState(() => ({
    goal_name: savings?.goal_name || known?.goal_name || '',
    goal_amount: moneyInput(savings?.goal_amount ?? known?.goal_amount),
    save_percent: known?.save_percent ?? '',
    term_months: known?.term_months ?? '',
    paused: savings?.paused ?? known?.paused ?? false,
    quiet_from: known?.quiet_from ?? '23:00',
    quiet_to: known?.quiet_to ?? '09:00',
    timezone: known?.timezone ?? 'Europe/Moscow',
    min_income: moneyInput(known?.min_income ?? 0),
    min_transfer: moneyInput(known?.min_transfer ?? 10000),
  }));
  const bind = (name) => ({ value: form[name], onChange: (event) => setForm({ ...form, [name]: event.target.value }) });
  async function submit(event) {
    event.preventDefault();
    if (mutation.isPending) return;
    setError(null);
    try {
      if (!form.goal_name.trim()) throw new Error('Введите название цели.');
      const body = {
        ...form,
        goal_name: form.goal_name.trim(),
        goal_amount: toKopecks(form.goal_amount),
        save_percent: Number(form.save_percent),
        term_months: Number(form.term_months),
        min_income: toKopecks(form.min_income),
        min_transfer: toKopecks(form.min_transfer),
      };
      await mutation.mutateAsync(body);
      toast(savings ? 'Правило сохранено.' : 'Копилка подключена.');
      onSaved({ created: !savings });
    } catch (err) {
      if (err.status === 409) {
        clearOnboarding(cache);
        toast(err.message);
        onRequiresOnboarding();
      } else setError(err);
    }
  }
  return <form onSubmit={submit}>
    <Modal.Body>
      {!savings && <p className="form-text mt-0 mb-3">Шаг 2 из 2. Анкета одобрена — укажите цель и правило накопления.</p>}
      {!known && savings && <p className="form-text mt-0 mb-3">Проверьте все параметры: сохранение заменит прежнее правило.</p>}
      <fieldset disabled={mutation.isPending}><div className="row g-3">
        <Field label="Название цели" maxLength={80} required {...bind('goal_name')} />
        <Field label="Сумма цели, ₽" className="col-12 col-sm-6" type="number" min="0.01" step="0.01" required {...bind('goal_amount')} />
        <Field label="Откладывать, %" className="col-12 col-sm-6" type="number" min="1" max="50" step="1" required {...bind('save_percent')} />
        <Field label="Срок" required {...bind('term_months')}><option value="" disabled>Выберите срок</option><option value="6">6 месяцев</option><option value="12">12 месяцев</option><option value="24">24 месяца</option></Field>
        <Checkbox label="Поставить копилку на паузу" checked={form.paused} onChange={(e) => setForm({ ...form, paused: e.target.checked })} />
        <Field label="Тихие часы с" className="col-6" type="time" required {...bind('quiet_from')} />
        <Field label="Тихие часы до" className="col-6" type="time" required {...bind('quiet_to')} />
        <div className="col-12"><details className="form-advanced"><summary>Дополнительные параметры</summary><div className="row g-3 mt-1">
          <Field label="Минимальный доход для пополнения, ₽" type="number" min="0" step="0.01" required {...bind('min_income')} />
          <Field label="Минимальный перевод, ₽" type="number" min="100" step="0.01" required {...bind('min_transfer')} />
          <Field label="Часовой пояс" required {...bind('timezone')} />
        </div></details></div>
      </div></fieldset>
      <InlineError error={error} />
    </Modal.Body>
    <Modal.Footer><AppButton type="submit" pending={mutation.isPending}>{savings ? 'Сохранить' : 'Подключить копилку'}</AppButton></Modal.Footer>
  </form>;
}

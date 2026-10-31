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
  const [quietEnabled, setQuietEnabled] = useState(() => Boolean(known && known.quiet_from !== known.quiet_to));
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
        quiet_from: quietEnabled ? form.quiet_from : '00:00',
        quiet_to: quietEnabled ? form.quiet_to : '00:00',
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
        <Field label="Название цели" placeholder="Например, подушка безопасности" maxLength={80} required {...bind('goal_name')} />
        <Field label="Сумма цели, ₽" className="col-12 col-sm-6" type="number" min="0.01" step="0.01" required {...bind('goal_amount')} />
        <Field label="Откладывать, %" placeholder="Например, 10" className="col-12 col-sm-6" type="number" min="1" max="50" step="1" required {...bind('save_percent')} />
        <Field label="Срок" required {...bind('term_months')}><option value="" disabled>Выберите срок</option><option value="6">6 месяцев</option><option value="12">12 месяцев</option><option value="24">24 месяца</option></Field>
        <Checkbox label="Поставить копилку на паузу" checked={form.paused} onChange={(e) => setForm({ ...form, paused: e.target.checked })} />
        <Checkbox label="Не беспокоить в тихие часы" hint="Откладывать уведомления о решениях до окончания выбранного времени." checked={quietEnabled} onChange={(e) => setQuietEnabled(e.target.checked)} />
        {quietEnabled && <><Field label="Тихие часы с" className="col-6" type="time" required {...bind('quiet_from')} />
        <Field label="Тихие часы до" className="col-6" type="time" required {...bind('quiet_to')} /></>}
        <div className="col-12"><details className="form-advanced"><summary>Дополнительные параметры</summary><div className="row g-3 mt-1">
          <Field label="Минимальный доход для пополнения, ₽" type="number" min="0" step="0.01" required {...bind('min_income')} />
          <Field label="Минимальный перевод, ₽" type="number" min="100" step="0.01" required {...bind('min_transfer')} />
          <Field label="Часовой пояс (для тихих часов и времени уведомлений)" required {...bind('timezone')}>
            {[["Europe/Kaliningrad", "Калининград · UTC+2"], ["Europe/Moscow", "Москва · UTC+3"], ["Europe/Samara", "Самара · UTC+4"], ["Asia/Yekaterinburg", "Екатеринбург · UTC+5"], ["Asia/Omsk", "Омск · UTC+6"], ["Asia/Krasnoyarsk", "Красноярск · UTC+7"], ["Asia/Irkutsk", "Иркутск · UTC+8"], ["Asia/Yakutsk", "Якутск · UTC+9"], ["Asia/Vladivostok", "Владивосток · UTC+10"], ["Asia/Magadan", "Магадан · UTC+11"], ["Asia/Kamchatka", "Камчатка · UTC+12"]].map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            {!["Europe/Kaliningrad", "Europe/Moscow", "Europe/Samara", "Asia/Yekaterinburg", "Asia/Omsk", "Asia/Krasnoyarsk", "Asia/Irkutsk", "Asia/Yakutsk", "Asia/Vladivostok", "Asia/Magadan", "Asia/Kamchatka"].includes(form.timezone) && <option value={form.timezone}>{form.timezone}</option>}
          </Field>
        </div></details></div>
      </div></fieldset>
      <InlineError error={error} />
    </Modal.Body>
    <Modal.Footer><AppButton type="submit" pending={mutation.isPending}>{savings ? 'Сохранить' : 'Подключить копилку'}</AppButton></Modal.Footer>
  </form>;
}

import { useState } from 'react';
import { useSubmitOnboarding } from './hooks';
import { Field, Checkbox } from '../../shared/ui/FormFields';
import AppButton from '../../shared/ui/AppButton';
import { InlineError } from '../../shared/ui/States';
import { toKopecks } from '../../shared/utils/money';

export default function OnboardingForm({ hasSavings, onChecked }) {
  const mutation = useSubmitOnboarding();
  const [error, setError] = useState(null);
  const [form, setForm] = useState({ birth_date: '', monthly_income: '', income_type: 'mixed', income_stability: 'irregular', income_months: '', has_loans: false, monthly_payments: '', data_consent: false, analysis_consent: false });
  const bind = (name) => ({ value: form[name], onChange: (e) => setForm({ ...form, [name]: e.target.value }) });
  const check = (name) => ({ checked: form[name], onChange: (e) => setForm({ ...form, [name]: e.target.checked }) });
  async function submit(event) {
    event.preventDefault();
    if (mutation.isPending) return;
    setError(null);
    try {
      await mutation.mutateAsync({ ...form, monthly_income: toKopecks(form.monthly_income), monthly_payments: toKopecks(form.monthly_payments), income_months: Number(form.income_months) });
      onChecked?.();
    } catch (err) { setError(err); }
  }
  return <form onSubmit={submit}><fieldset disabled={mutation.isPending}><div className="row g-3">
    <Field label="Дата рождения" className="col-12 col-md-6" type="date" required {...bind('birth_date')} />
    <Field label="Доход в месяц, ₽" className="col-12 col-md-6" type="number" min="0" step="0.01" required {...bind('monthly_income')} />
    <Field label="Источник дохода" className="col-12 col-md-6" {...bind('income_type')}><option value="mixed">Разные источники</option><option value="salary">Зарплата</option><option value="scholarship">Стипендия</option><option value="freelance">Фриланс</option><option value="none">Дохода пока нет</option></Field>
    <Field label="Регулярность дохода" className="col-12 col-md-6" {...bind('income_stability')}><option value="irregular">Нерегулярный</option><option value="stable">Стабильный</option></Field>
    <Field label="Сколько месяцев есть доход" className="col-12 col-md-6" type="number" min="0" step="1" required {...bind('income_months')} />
    <Field label="Обязательные платежи в месяц, ₽" className="col-12 col-md-6" type="number" min="0" step="0.01" required {...bind('monthly_payments')} />
    <Checkbox label="Есть кредиты, рассрочки или займы" hint="Их платежи укажите в поле обязательных платежей выше." {...check('has_loans')} />
    <Checkbox label="Согласие на обработку персональных данных" required {...check('data_consent')} />
    <Checkbox label="Согласие на анализ доходов и расходов" hint="Без согласия копилка работает в простом режиме: откладывает фиксированный процент без анализа операций и вопросов." {...check('analysis_consent')} />
  </div></fieldset><InlineError error={error} /><AppButton className="mt-3" type="submit" pending={mutation.isPending}>{hasSavings ? 'Обновить анкету' : 'Проверить анкету'}</AppButton></form>;
}

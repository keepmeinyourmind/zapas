import { Link, useOutletContext } from 'react-router-dom';
import SavingsCard from '../features/savings/SavingsCard';
import { useKnownRule, useSaveRule } from '../features/rule/hooks';
import { useToast } from '../app/providers';
import { formatMoney } from '../shared/utils/money';
import AppButton from '../shared/ui/AppButton';
import { LoadingState, ErrorState, InlineError } from '../shared/ui/States';
import ConnectionNotice from '../features/onboarding/ConnectionNotice';

export default function SavingsPage() {
  const { savings, onboarding, onRule, onWithdraw } = useOutletContext();
  const { data: rule } = useKnownRule();
  const pause = useSaveRule(true);
  const toast = useToast();
  async function togglePause() {
    if (!savings.data || pause.isPending) return;
    try { await pause.mutateAsync({ paused: !savings.data.paused }); toast('Статус копилки изменён.'); }
    catch { /* Ошибка отображается рядом с кнопкой действия. */ }
  }
  return <section className="view" id="view-savings">
    <div className="mb-4"><h1 className="page-title">Копилка и правило</h1><p className="page-description">Цель и параметры накопления</p>{savings.data && (onboarding?.eligible === false ? <Link className="reference-button mt-3 text-decoration-none" to="/onboarding">Обновить анкету</Link> : <AppButton className="mt-3" onClick={onRule} disabled={savings.isError}>Изменить правило</AppButton>)}</div>
    {savings.isPending ? <LoadingState /> : savings.isError && !savings.data ? savings.error.status === 404 ? <ConnectionNotice onboarding={onboarding} onRule={onRule} /> : <ErrorState error={savings.error} retry={savings.refetch} /> : <div className="row g-3">
      <div className="col-12"><SavingsCard savings={savings.data} rule={rule} onWithdraw={onWithdraw} /></div>
      <div className="col-12"><article className="card h-100"><div className="card-body p-4"><h2 className="section-title mb-3">Безопасное накопление</h2>
        <div className="list-group list-group-flush">
          <div className="list-group-item d-flex justify-content-between gap-3"><span>Минимальный доход</span><strong>{formatMoney(rule?.min_income)}</strong></div>
          <div className="list-group-item d-flex justify-content-between gap-3"><span>Тихие часы</span><strong>{rule ? `${rule.quiet_from}–${rule.quiet_to}` : '—'}</strong></div>
          <div className="list-group-item d-flex justify-content-between gap-3"><span>Минимальный перевод</span><strong>{formatMoney(rule?.min_transfer)}</strong></div>
        </div>
        {!rule && <p className="form-text mt-3 mb-0">Параметры появятся после сохранения правила или изменения паузы.</p>}
        <AppButton className="mt-4" onClick={togglePause} pending={pause.isPending} disabled={savings.isError}>{savings.data.paused ? 'Возобновить копилку' : 'Поставить на паузу'}</AppButton><InlineError error={pause.error} />
        {savings.data.mode && <p className="form-text mt-3 mb-0">{savings.data.mode === 'simple' ? 'Простой режим: фиксированный процент от поступлений.' : 'Умный режим: с анализом доходов и расходов.'}</p>}
      </div></article></div>
    </div>}
    {savings.isError && savings.data && <ErrorState error={savings.error} retry={savings.refetch} />}
  </section>;
}

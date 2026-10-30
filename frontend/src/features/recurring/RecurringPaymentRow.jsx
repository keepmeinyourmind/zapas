import { useSetExclusion } from './hooks';
import { useToast } from '../../app/providers';
import { formatMoney } from '../../shared/utils/money';
import { formatDate } from '../../shared/utils/date';
import { InlineError } from '../../shared/ui/States';
export default function RecurringPaymentRow({ payment }) {
  const mutation = useSetExclusion();
  const toast = useToast();
  async function toggle(excluded) {
    if (mutation.isPending) return;
    try { await mutation.mutateAsync({ counterparty: payment.counterparty, excluded }); toast('Параметры обновлены.'); }
    catch { /* Ошибка отображается рядом с этим платежом. */ }
  }
  return <div className="recurring-row py-3"><div className="row align-items-center g-3">
    <div className="col"><div className="transaction-name">{payment.counterparty}</div><div className="transaction-meta">{formatMoney(payment.amount)} · {payment.day_of_month}-го числа · последнее списание {formatDate(payment.last_at)}</div></div>
    <div className="col-auto text-secondary small recurring-label">{payment.excluded ? 'Необязательный' : 'Обязательный'}</div>
    <div className="col-auto"><div className="form-check form-switch mb-0"><input className="form-check-input recurring-toggle" type="checkbox" role="switch" checked={!payment.excluded} disabled={mutation.isPending} onChange={(e) => toggle(!e.target.checked)} aria-label={`Учитывать «${payment.counterparty}» как обязательный платёж`} /></div></div>
  </div><InlineError error={mutation.error} /></div>;
}

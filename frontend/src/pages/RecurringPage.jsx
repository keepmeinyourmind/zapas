import { useRecurring } from '../features/recurring/hooks';
import RecurringPaymentRow from '../features/recurring/RecurringPaymentRow';
import { LoadingState, ErrorState, EmptyState } from '../shared/ui/States';
export default function RecurringPage() {
  const query = useRecurring();
  return <section className="view" id="view-recurring"><div className="mb-4"><h1 className="page-title">Регулярные платежи</h1><p className="page-description">Отметьте платежи, которые нужно учитывать как обязательные</p></div>
    <article className="card"><div className="card-body p-3 p-lg-4">{query.isPending ? <LoadingState /> : query.isError ? <ErrorState error={query.error} retry={query.refetch} /> : query.data.length ? query.data.map((payment) => <RecurringPaymentRow payment={payment} key={payment.counterparty} />) : <EmptyState>Регулярные платежи пока не найдены.</EmptyState>}</div></article>
    <p className="form-text mt-3">Необязательные платежи не резервируют деньги при расчёте безопасной суммы.</p>
  </section>;
}

import { useState } from 'react';
import { useTransactions } from '../features/transactions/hooks';
import TransactionList, { isCredit } from '../features/transactions/TransactionList';
import { LoadingState, ErrorState } from '../shared/ui/States';
import AppButton from '../shared/ui/AppButton';
export default function TransactionsPage() {
  const [kind, setKind] = useState('expense');
  const [limit, setLimit] = useState(50);
  const query = useTransactions(limit);
  return <section className="view" id="view-transactions">
    <div className="mb-4"><h1 className="page-title">Операции</h1><p className="page-description">Доходы и расходы, новые сверху</p></div>
    <div className="d-flex gap-2 mb-3 filter-row" role="group" aria-label="Фильтр операций">{[['expense', 'Расходы'], ['income', 'Зачисления']].map(([value, text]) => <button className={`btn filter-chip${kind === value ? ' active' : ''}`} type="button" key={value} aria-pressed={kind === value} onClick={() => setKind(value)}>{text}</button>)}</div>
    <article className="card"><div className="card-body p-3 p-lg-4">{query.isPending ? <LoadingState /> : query.isError ? <ErrorState error={query.error} retry={query.refetch} /> : <>
      <TransactionList transactions={query.data.filter((tx) => isCredit(tx) === (kind === 'income'))} />
      {query.data.length >= limit && (limit < 200 ? <AppButton className="mt-3" onClick={() => setLimit(Math.min(200, limit + 50))}>Показать больше</AppButton> : <p className="form-text mt-3 mb-0">Показаны последние 200 операций.</p>)}
    </>}</div></article>
  </section>;
}

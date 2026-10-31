import { useState } from 'react';
import { useTransactions, useOperations } from '../features/transactions/hooks';
import OperationList from '../features/transactions/OperationList';
import TransactionList, { isCredit } from '../features/transactions/TransactionList';
import { LoadingState, ErrorState } from '../shared/ui/States';
import AppButton from '../shared/ui/AppButton';
export default function TransactionsPage() {
  const [kind, setKind] = useState('savings');
  const [limit, setLimit] = useState(50);
  const transactions = useTransactions(limit, { enabled: kind !== 'savings' });
  const operations = useOperations(Math.min(limit, 100), { enabled: kind === 'savings' });
  const query = kind === 'savings' ? operations : transactions;
  const maxLimit = kind === 'savings' ? 100 : 200;
  return <section className="view" id="view-transactions">
    <div className="mb-4"><h1 className="page-title">Операции</h1><p className="page-description">Движения копилки и операции карты, новые сверху</p></div>
    <div className="d-flex gap-2 mb-3 filter-row" role="group" aria-label="Фильтр операций">{[['savings', 'Копилка'], ['expense', 'Расходы'], ['income', 'Зачисления']].map(([value, text]) => <button className={`btn filter-chip${kind === value ? ' active' : ''}`} type="button" key={value} aria-pressed={kind === value} onClick={() => { setKind(value); setLimit(50); }}>{text}</button>)}</div>
    <article className="card"><div className="card-body p-3 p-lg-4">{query.isPending ? <LoadingState /> : query.isError ? <ErrorState error={query.error} retry={query.refetch} /> : <>
      {kind === 'savings' ? <OperationList operations={query.data} /> : <TransactionList transactions={query.data.filter((tx) => isCredit(tx) === (kind === 'income'))} />}
      {query.data.length >= limit && (limit < maxLimit ? <AppButton className="mt-3" onClick={() => setLimit(Math.min(maxLimit, limit + 50))}>Показать больше</AppButton> : <p className="form-text mt-3 mb-0">Показаны последние {maxLimit} операций.</p>)}
    </>}</div></article>
  </section>;
}

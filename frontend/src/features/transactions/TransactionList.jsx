import { formatMoney } from '../../shared/utils/money';
import { dateKey, formatDate } from '../../shared/utils/date';
import { EmptyState } from '../../shared/ui/States';

const incomeNames = { salary: 'Зарплата', advance: 'Аванс', bonus: 'Премия', scholarship: 'Стипендия', freelance: 'Фриланс' };
const typeNames = { incoming: 'Зачисление', outgoing: 'Расходы', refund: 'Возврат', internal: 'Перевод между счетами', loan: 'Кредитные средства' };
export const isCredit = (transaction) => transaction.type !== 'outgoing';

export default function TransactionList({ transactions = [] }) {
  if (!transactions.length) return <EmptyState>Операций пока нет.</EmptyState>;
  const groups = new Map();
  [...transactions].sort((a, b) => new Date(b.occurred_at) - new Date(a.occurred_at)).forEach((tx) => {
    const key = dateKey(tx.occurred_at);
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(tx);
  });
  return <div className="money-history">{[...groups].map(([key, transactions]) => <section className="money-history-group" key={key}>
    <div className="money-history-date">{formatDate(transactions[0].occurred_at)}</div>
    {transactions.map((tx) => <div className="money-history-row" key={tx.id}><div className="money-history-copy"><div className="transaction-name">{tx.counterparty || incomeNames[tx.income_kind] || typeNames[tx.type]}</div><div className="transaction-meta">{incomeNames[tx.income_kind] || typeNames[tx.type] || 'Операция по карте'}</div></div><span className={`transaction-amount${isCredit(tx) ? ' positive' : ''}`}>{isCredit(tx) ? '+' : '−'}{formatMoney(tx.amount)}</span></div>)}
  </section>)}</div>;
}

import TransactionList from './TransactionList';

export default function OperationList({ operations = [] }) {
  return <TransactionList transactions={operations.map((op) => ({
    id: `${op.type}-${op.id}`,
    type: op.type === 'transfer_in' ? 'savings_in' : 'savings_out',
    counterparty: op.type === 'transfer_in' ? 'Пополнение копилки' : 'Снятие из копилки',
    amount: op.amount,
    occurred_at: op.created_at,
  }))} />;
}

import { useState } from 'react';
import DepositModal from '../features/savings/DepositModal';
import { Link, useOutletContext } from 'react-router-dom';
import { Navigation } from '../layouts/AppLayout';
import SavingsCard from '../features/savings/SavingsCard';
import BalanceCards from '../features/savings/BalanceCards';
import { useDecisions } from '../features/decisions/hooks';
import { useTransactions } from '../features/transactions/hooks';
import TransactionList from '../features/transactions/TransactionList';
import { LoadingState, ErrorState } from '../shared/ui/States';
import ConnectionNotice from '../features/onboarding/ConnectionNotice';

export default function DashboardPage() {
  const [showDepositModal, setShowDepositModal] = useState(false);
  const { savings, onboarding, onRule, onWithdraw, onExplain } = useOutletContext();
  const connected = Boolean(savings.data);
  const decisions = useDecisions({}, { enabled: connected });
  const pending = useDecisions({ status: 'pending' }, { enabled: connected });
  const transactions = useTransactions(50, { enabled: connected });
  const latest = decisions.data?.[0];
  // Объяснение относится к прошлому решению; в простом режиме безопасная сумма не рассчитывается.
  const explanationDecision = savings.data?.mode === 'smart' && latest?.explanation ? latest : null;
  const needsAction = Boolean(
    (!savings.isError && !savings.data?.arrested && !savings.data?.paused &&
      pending.data?.some((item) => item.status === 'pending' && ['ask', 'confirm_income'].includes(item.action))) ||
    latest?.action === 'suggest_raise'
  );
  return <section className="view active" id="view-home"><div className="home-column"><div className="row g-3">
    <div className="col-12">{savings.isPending ? <LoadingState /> : savings.isError && !savings.data ? savings.error.status === 404 ? <ConnectionNotice onboarding={onboarding} onRule={onRule} /> : <ErrorState error={savings.error} retry={savings.refetch} /> : <SavingsCard savings={savings.data} overview onRule={onRule} onWithdraw={onWithdraw} onDeposit={() => setShowDepositModal(true)} />}</div>
    <div className="col-12"><Navigation connected={connected} hasDecisions={needsAction} /></div>
    {savings.data && <BalanceCards savings={savings.data} decision={explanationDecision} onExplain={onExplain} />}
    {savings.isError && savings.data && <div className="col-12"><ErrorState error={savings.error} retry={savings.refetch} /></div>}
    {connected && decisions.isError && <div className="col-12"><ErrorState error={decisions.error} retry={decisions.refetch} /></div>}
    {connected && pending.isError && !decisions.isError && <div className="col-12"><ErrorState error={pending.error} retry={pending.refetch} /></div>}
    {connected && <div className="col-12"><article className="card"><div className="card-body p-3 p-lg-4"><div className="d-flex justify-content-between align-items-center gap-3 mb-3"><h2 className="section-title">Последние операции</h2></div>
      {transactions.isPending ? <LoadingState /> : transactions.isError ? <ErrorState error={transactions.error} retry={transactions.refetch} /> : <TransactionList transactions={transactions.data.slice(0, 3)} />}
      <Link className="reference-button mt-3 text-decoration-none" to="/transactions">Перейти в историю</Link>
    </div></article></div>}
  </div></div><DepositModal show={showDepositModal} onHide={() => setShowDepositModal(false)} onSuccess={() => savings.refetch()} /></section>;
}

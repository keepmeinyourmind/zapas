import InfoButton from '../../shared/ui/InfoButton';
import { formatMoney } from '../../shared/utils/money';

export default function BalanceCards({ savings, decision, onExplain }) {
  const safeAmount = savings.mode === 'smart' ? savings.safe_amount : null;
  return <>
    <div className="col-6"><article className="card stat-card h-100"><div className="card-body p-3"><div className="stat-label">Баланс карты</div><strong className="stat-value">{formatMoney(savings.card_balance)}</strong><div className="stat-note">собственные средства</div></div></article></div>
    <div className="col-6"><article className="card stat-card h-100"><div className="card-body p-3"><div className="stat-label stat-label-with-info"><span>Безопасно отложить</span><InfoButton disabled={!decision?.explanation} onClick={() => onExplain(decision)} /></div><strong className="stat-value text-success">{formatMoney(safeAmount)}</strong><div className="stat-note">{safeAmount != null ? 'по текущему остатку карты' : 'В простом режиме не рассчитывается'}</div></div></article></div>
  </>;
}

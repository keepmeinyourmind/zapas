import DecisionActions from './DecisionActions';
import { formatMoney } from '../../shared/utils/money';
import { formatDate } from '../../shared/utils/date';

const statuses = { pending: 'Ожидает ответа', queued: 'В очереди', executed: 'Выполнено', skipped: 'Пропущено', expired: 'Время ответа истекло' };
function title(decision) {
  if (decision.action === 'confirm_income') return 'Это ваш доход?';
  if (decision.status === 'skipped' || decision.action === 'skip') return 'Пополнение пропущено';
  if (decision.status === 'executed') return `Отложено ${formatMoney(decision.amount)}`;
  if (decision.status === 'expired') return 'Время на ответ вышло';
  return `Предложение отложить ${formatMoney(decision.amount)}`;
}
export default function DecisionCard({ decision, blocked, onExplain, onRule }) {
  const completed = !['pending', 'queued'].includes(decision.status);
  return <div className="decision-row py-3">
    <div className="d-flex flex-wrap justify-content-between align-items-center gap-2"><strong>{title(decision)}</strong><span className={`decision-status${completed ? ' done' : ''}`}>{statuses[decision.status] || decision.status}</span></div>
    <p className="text-secondary small mt-2 mb-3">{decision.reason}</p>
    {decision.status === 'queued' && <p className="form-text">Сначала ответьте на предыдущий вопрос.</p>}
    <DecisionActions key={`${decision.id}-${decision.action}-${decision.status}`} decision={decision} blocked={blocked} />
    {decision.action === 'suggest_raise' && onRule && <button className="decision-action decision-action-primary" type="button" onClick={onRule}>Изменить процент</button>}
    <div className="d-flex flex-wrap align-items-center justify-content-between gap-2 mt-3">
      <small className="text-secondary">{formatDate(decision.created_at, true)}{decision.status === 'pending' && decision.expires_at && <> · ответ до {formatDate(decision.expires_at, true)}</>}</small>
      {decision.explanation && <button className="section-link" type="button" onClick={() => onExplain(decision)}>Подробности расчёта</button>}
    </div>
  </div>;
}

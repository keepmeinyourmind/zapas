import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useDecisions } from './hooks';
import { config } from '../../app/config';

const key = `zapas:notifications:${config.clientId}`;
function readDismissed() {
  try {
    const value = JSON.parse(sessionStorage.getItem(key));
    return Array.isArray(value) ? value : [];
  } catch { return []; }
}

export default function DecisionNotifications({ savings }) {
  const { pathname } = useLocation();
  const { data = [] } = useDecisions({}, { enabled: Boolean(savings.data) });
  const [dismissed, setDismissed] = useState(readDismissed);
  const decision = data.find((item) => {
    const actionable = item.status === 'pending' && ['ask', 'confirm_income'].includes(item.action) &&
      item.id === savings.data?.pending_decision_id && !savings.data?.paused && !savings.data?.arrested;
    const raise = item.action === 'suggest_raise' && item.status === 'executed' && item.id === data[0]?.id;
    return (actionable || raise) && (!item.notify_at || new Date(item.notify_at) <= new Date()) &&
      !dismissed.includes(`${item.id}:${item.action}:${item.status}`);
  });
  if (!decision || pathname === '/decisions' || savings.isError) return null;
  function dismiss() {
    const next = [...dismissed, `${decision.id}:${decision.action}:${decision.status}`];
    setDismissed(next);
    try { sessionStorage.setItem(key, JSON.stringify(next)); } catch { /* Доступно до перезагрузки. */ }
  }
  return <aside className="decision-push" aria-label="Уведомление о решении" role="status">
    <div className="d-flex justify-content-between gap-3"><strong>{decision.action === 'suggest_raise' ? 'Можно повысить процент' : 'Нужен ваш ответ'}</strong><button className="section-link" type="button" aria-label="Закрыть уведомление" onClick={dismiss}>×</button></div>
    <p className="small mt-2 mb-3">{decision.reason}</p>
    <Link className="reference-button text-decoration-none" to="/decisions">Перейти к решению</Link>
  </aside>;
}

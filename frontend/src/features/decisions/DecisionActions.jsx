import { useRef, useState } from 'react';
import { useAnswerDecision } from './hooks';
import { useToast } from '../../app/providers';
import { formatMoney, toKopecks } from '../../shared/utils/money';
import { InlineError } from '../../shared/ui/States';

export default function DecisionActions({ decision, blocked }) {
  const mutation = useAnswerDecision();
  const [error, setError] = useState(null);
  const [custom, setCustom] = useState('');
  const busy = useRef(false);
  const toast = useToast();
  if (decision.status !== 'pending' || !['ask', 'confirm_income'].includes(decision.action)) return null;
  const options = [...new Set([...(decision.options || []), decision.amount].filter((v) => v > 0))];
  async function answer(body) {
    if (busy.current || blocked) return;
    busy.current = true;
    setError(null);
    try {
      const result = await mutation.mutateAsync({ id: decision.id, ...body });
      toast(result.reason || 'Ответ принят.');
      setCustom('');
    } catch (err) { setError(err); }
    finally { busy.current = false; }
  }
  const button = (text, body, primary = false) => <button key={text} className={`decision-action decision-action-${primary ? 'primary' : 'secondary'}`} type="button" onClick={() => answer(body)} disabled={mutation.isPending || blocked}>{text}</button>;
  return <>
    <div className="decision-actions" aria-busy={mutation.isPending}>
      {decision.action === 'confirm_income' ? <>
        {button('Да, это мой доход', { choice: 'yes' }, true)}
        {button('Нет, это не мой доход', { choice: 'no' })}
      </> : <>
        {options.map((amount) => button(`Отложить ${formatMoney(amount)}`, { choice: 'amount', amount }, true))}
        {button('Пропустить', { choice: 'skip' })}
      </>}
    </div>
    {decision.action === 'ask' && <details className="form-advanced mt-3"><summary>Другая сумма</summary><form className="d-flex flex-wrap gap-2 mt-2" onSubmit={(e) => {
      e.preventDefault();
      try { answer({ choice: 'amount', amount: toKopecks(custom) }); } catch (err) { setError(err); }
    }}><input className="form-control custom-amount" type="number" step="0.01" min="0.01" max={options.length ? Math.max(...options) / 100 : undefined} aria-label="Своя сумма, ₽" placeholder="Сумма, ₽" required value={custom} onChange={(e) => setCustom(e.target.value)} disabled={mutation.isPending || blocked} /><button className="decision-action decision-action-primary" type="submit" disabled={mutation.isPending || blocked}>Отложить</button></form></details>}
    {blocked && <p className="form-text mt-2">Ответ сейчас недоступен. Проверьте статус копилки.</p>}
    <InlineError error={error} />
  </>;
}

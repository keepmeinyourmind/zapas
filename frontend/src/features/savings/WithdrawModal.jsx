import { useRef, useState } from 'react';
import Modal from 'react-bootstrap/Modal';
import { useWithdraw } from './hooks';
import { useToast } from '../../app/providers';
import AppButton from '../../shared/ui/AppButton';
import { Field } from '../../shared/ui/FormFields';
import { InlineError } from '../../shared/ui/States';
import { formatMoney, moneyInput, toKopecks } from '../../shared/utils/money';
import { readWithdrawal, saveWithdrawal } from '../../shared/utils/withdrawalAttempt';
import { config } from '../../app/config';

function WithdrawForm({ savings, onHide }) {
  const [amount, setAmount] = useState(() => moneyInput(readWithdrawal(config.clientId)?.amount));
  const [error, setError] = useState(null);
  const attempt = useRef(readWithdrawal(config.clientId));
  const inFlight = useRef(false);
  const mutation = useWithdraw();
  const toast = useToast();
  const blocked = !attempt.current && (savings?.arrested || !savings || savings.balance <= 0);
  async function submit(event) {
    event.preventDefault();
    if (inFlight.current || blocked) return;
    setError(null);
    try {
      const value = attempt.current?.amount ?? toKopecks(amount);
      if (!attempt.current && (value <= 0 || value > savings.balance)) throw new Error('Введите сумму в пределах баланса копилки.');
      // При потере ответа сохраняем сумму и ID: повторный запрос относится к той же операции.
      if (!attempt.current) {
        attempt.current = { id: crypto.randomUUID(), amount: value };
        saveWithdrawal(config.clientId, attempt.current);
      }
      inFlight.current = true;
      await mutation.mutateAsync(attempt.current);
      saveWithdrawal(config.clientId, null);
      toast('Деньги переведены на карту.');
      onHide();
    } catch (err) {
      if (err.status >= 400 && err.status < 500) {
        attempt.current = null;
        saveWithdrawal(config.clientId, null);
      }
      setError(err);
    } finally { inFlight.current = false; }
  }
  return <form onSubmit={submit}>
    <Modal.Body>
      <Field label="Сумма, ₽" type="number" min="0.01" step="0.01" max={moneyInput(savings?.balance)} required value={amount} disabled={mutation.isPending || Boolean(attempt.current) || blocked} onChange={(e) => setAmount(e.target.value)} />
      <div className="form-text mt-2">Доступно: {formatMoney(savings?.balance)}</div>
      {savings?.arrested && <p className="form-error mt-2">Снятие недоступно: на счёт наложены ограничения.</p>}
      {attempt.current && !mutation.isPending && <p className="form-text">Подтверждение не получено. Повторите запрос с той же суммой, чтобы избежать повторного снятия.</p>}
      <InlineError error={error} />
    </Modal.Body>
    <Modal.Footer><AppButton type="submit" pending={mutation.isPending} disabled={blocked}>Подтвердить</AppButton></Modal.Footer>
  </form>;
}

export default function WithdrawModal({ show, onHide, savings }) {
  return <Modal show={show} onHide={onHide} centered size="sm" aria-labelledby="withdraw-title">
    <Modal.Header closeButton closeLabel="Закрыть"><Modal.Title as="h2" id="withdraw-title">Снять из копилки</Modal.Title></Modal.Header>
    {show && <WithdrawForm savings={savings} onHide={onHide} />}
  </Modal>;
}

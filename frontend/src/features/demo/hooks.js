import { useEffect, useRef, useState } from 'react';
import { useClientMutation } from '../../shared/api/hooks';
import { config } from '../../app/config';
import { readBankRequest, storeBankRequest } from './pendingRequest';

export function useBankSubmission(operation, mutationFn, onPendingChange) {
  const [attempt, setAttempt] = useState(() => readBankRequest(operation));
  const [error, setError] = useState(null);
  const attemptRef = useRef(attempt);
  const inFlight = useRef(false);
  const mutation = useClientMutation(mutationFn, { mutationKey: ['demo', operation] });
  useEffect(() => { onPendingChange?.(Boolean(attempt)); }, [attempt, onPendingChange]);

  function updateAttempt(body) {
    attemptRef.current = body;
    storeBankRequest(operation, body);
    setAttempt(body);
  }

  async function send(makeBody) {
    if (inFlight.current) return;
    inFlight.current = true;
    setError(null);
    mutation.reset();
    try {
      if (!attemptRef.current) updateAttempt({ ...makeBody(), client_id: config.clientId, id: crypto.randomUUID() });
      const result = await mutation.mutateAsync(attemptRef.current);
      updateAttempt(null);
      return result;
    } catch (err) {
      // При потере подтверждения запрос мог уже выполниться. Сохраняем полное тело запроса для повтора.
      if (err.status >= 400 && err.status < 500) updateAttempt(null);
      setError(err);
    } finally { inFlight.current = false; }
  }
  return { attempt, error, send, mutation };
}

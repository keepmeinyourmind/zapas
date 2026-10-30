const attempts = new Map();
const keyFor = (clientId) => `zapas:withdrawal:${clientId}`;

export function readWithdrawal(clientId) {
  if (attempts.has(clientId)) return attempts.get(clientId);
  try {
    const value = JSON.parse(sessionStorage.getItem(keyFor(clientId)));
    if (value && typeof value.id === 'string' && Number.isSafeInteger(value.amount) && value.amount > 0) {
      attempts.set(clientId, value);
      return value;
    }
  } catch { /* Если хранилище недоступно, сохраняем попытку в памяти. */ }
  return null;
}

export function saveWithdrawal(clientId, attempt) {
  if (attempt) attempts.set(clientId, attempt);
  else attempts.delete(clientId);
  try {
    if (attempt) sessionStorage.setItem(keyFor(clientId), JSON.stringify(attempt));
    else sessionStorage.removeItem(keyFor(clientId));
  } catch { /* Закрытие модалки не должно удалять операцию без подтверждённого результата. */ }
}

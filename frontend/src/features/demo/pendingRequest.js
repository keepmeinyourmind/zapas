import { config } from '../../app/config';

const memory = new Map();
const storageKey = (operation) => `zapas:bank-request:${config.apiBaseUrl}:${config.clientId}:${operation}`;

export function readBankRequest(operation) {
  const key = storageKey(operation);
  if (memory.has(key)) return memory.get(key);
  try {
    const body = JSON.parse(sessionStorage.getItem(key));
    if (body?.client_id === config.clientId && typeof body.id === 'string' && body.id && typeof body.occurred_at === 'string') {
      memory.set(key, body);
      return body;
    }
  } catch { /* Хранилище браузера может быть недоступно. */ }
  return null;
}

export function storeBankRequest(operation, body) {
  const key = storageKey(operation);
  if (body) memory.set(key, body);
  else memory.delete(key);
  try {
    if (body) sessionStorage.setItem(key, JSON.stringify(body));
    else sessionStorage.removeItem(key);
  } catch { /* Если хранилище заблокировано, сохраняем в памяти запросы без подтверждённого результата. */ }
}

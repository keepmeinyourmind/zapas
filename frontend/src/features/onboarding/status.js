import { config } from '../../app/config';
import { resourceKey } from '../../shared/api/hooks';

const storageKey = `zapas:onboarding:${config.clientId}`;

// Локальный статус позволяет продолжить подключение; PUT /rule повторно проверяет допуск на сервере.
// Не сохраняем поля анкеты и персональные данные.
export function readOnboardingReceipt() {
  try {
    const receipt = JSON.parse(sessionStorage.getItem(storageKey));
    if (receipt?.client_id === config.clientId && typeof receipt.eligible === 'boolean') return receipt;
  } catch { /* В текущей вкладке можно продолжить работу и без доступа к хранилищу. */ }
  return null;
}

export function recordOnboarding(cache, result) {
  cache.setQueryData(resourceKey('onboarding'), result);
  try {
    sessionStorage.setItem(storageKey, JSON.stringify({ client_id: result.client_id, eligible: result.eligible, mode: result.mode }));
  } catch { /* Ответ сервера остаётся в кэше запросов до конца сеанса. */ }
}

export function clearOnboarding(cache) {
  cache.setQueryData(resourceKey('onboarding'), null);
  try { sessionStorage.removeItem(storageKey); } catch { /* Сохранённый статус недоступен. */ }
}

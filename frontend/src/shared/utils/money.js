const numberFormat = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 2 });

export function moneyNumber(kopecks) {
  return Number.isSafeInteger(kopecks) ? numberFormat.format(kopecks / 100) : '—';
}

export function formatMoney(kopecks) {
  return Number.isSafeInteger(kopecks) ? `${moneyNumber(kopecks)} ₽` : '—';
}

// Преобразуем десятичную строку, избегая погрешности умножения дробных чисел (например, 1.01 * 100).
export function toKopecks(value) {
  const text = String(value).trim().replace(',', '.');
  if (!/^\d+(?:\.\d{1,2})?$/.test(text)) throw new Error('Введите сумму с точностью до копейки.');
  const [rubles, fractional = ''] = text.split('.');
  const amount = Number(rubles) * 100 + Number(fractional.padEnd(2, '0'));
  if (!Number.isSafeInteger(amount)) throw new Error('Сумма слишком большая.');
  return amount;
}

export function moneyInput(kopecks) {
  return Number.isSafeInteger(kopecks) ? String(kopecks / 100) : '';
}

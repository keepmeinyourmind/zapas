export function decisionReason(decision) {
  if (decision.status === 'skipped' && decision.reason?.startsWith('Пока вы решали, на карте стало меньше денег')) {
    return 'На карте стало меньше денег.';
  }
  return decision.reason;
}

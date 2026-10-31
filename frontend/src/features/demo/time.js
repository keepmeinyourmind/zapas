export function moscowDateTime(value = new Date()) {
  const parts = Object.fromEntries(new Intl.DateTimeFormat('en-GB', {
    timeZone: 'Europe/Moscow', year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23',
  }).formatToParts(new Date(value)).map(({ type, value }) => [type, value]));
  return `${parts.year}-${parts.month}-${parts.day}T${parts.hour}:${parts.minute}:${parts.second}`;
}

export function eventTimestamp(value) {
  const date = new Date(`${value}+03:00`);
  if (!value || !Number.isFinite(date.getTime())) throw new Error('Укажите дату и время события.');
  return date.toISOString();
}

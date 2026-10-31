import { config } from '../../app/config';

export class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

export async function request(path, { body, signal, method = 'GET' } = {}) {
  let response;
  try {
    response = await fetch(`${config.apiBaseUrl}${path}`, {
      method,
      signal,
      headers: body === undefined ? { Accept: 'application/json' } : {
        Accept: 'application/json', 'Content-Type': 'application/json',
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch (error) {
    if (error.name === 'AbortError') throw error;
    throw new ApiError('Не удалось связаться с сервером. Проверьте соединение и повторите попытку.', 0);
  }
  const raw = await response.text();
  let data;
  try {
    data = raw ? JSON.parse(raw) : null;
  } catch {
    throw new ApiError('Сервер вернул неожиданный ответ. Повторите попытку.', response.status);
  }
  if (!response.ok) {
    throw new ApiError(data?.error || 'Не удалось выполнить запрос. Попробуйте ещё раз.', response.status);
  }
  return data;
}

export const clientPath = `/api/clients/${encodeURIComponent(config.clientId)}`;

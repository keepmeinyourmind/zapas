import { request } from '../../shared/api/client';

export const checkHealth = (signal) => request('/health', { signal });
export const createTransaction = (body) => request('/api/transactions', { method: 'POST', body });
export const createAccountEvent = (body) => request('/api/events/account', { method: 'POST', body });
export const runScenario = (name) => request(`/api/scenarios/${encodeURIComponent(name)}/run`, { method: 'POST' });
export const advanceTime = (hours) => request('/api/debug/advance-time', { method: 'POST', body: { hours } });

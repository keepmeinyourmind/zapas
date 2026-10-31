import { request, clientPath } from '../../shared/api/client';
export const getSavings = (signal) => request(`${clientPath}/savings`, { signal });
export const withdraw = (body) => request(`${clientPath}/savings/withdraw`, { method: 'POST', body });
export const deposit = (body) => request(`${clientPath}/savings/deposit`, { method: 'POST', body });
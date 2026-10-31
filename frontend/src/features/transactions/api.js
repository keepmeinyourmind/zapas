import { request, clientPath } from '../../shared/api/client';
export const getTransactions = (limit, signal) => request(`${clientPath}/transactions?limit=${limit}`, { signal });

export const getOperations = (limit, signal) => request(`${clientPath}/operations?limit=${limit}`, { signal });

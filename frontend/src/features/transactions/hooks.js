import { useQuery } from '@tanstack/react-query';
import { resourceKey } from '../../shared/api/hooks';
import { getTransactions, getOperations } from './api';
export const useTransactions = (limit = 50, { enabled = true } = {}) => useQuery({ queryKey: resourceKey('transactions', limit), queryFn: ({ signal }) => getTransactions(limit, signal), select: (data) => data || [], enabled });

export const useOperations = (limit = 50, { enabled = true } = {}) => useQuery({ queryKey: resourceKey('operations', limit), queryFn: ({ signal }) => getOperations(limit, signal), select: (data) => data || [], enabled, refetchInterval: 15000 });

import { useQuery } from '@tanstack/react-query';
import { resourceKey } from '../../shared/api/hooks';
import { getTransactions } from './api';
export const useTransactions = (limit = 50, { enabled = true } = {}) => useQuery({ queryKey: resourceKey('transactions', limit), queryFn: ({ signal }) => getTransactions(limit, signal), select: (data) => data || [], enabled });

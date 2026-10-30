import { useQuery } from '@tanstack/react-query';
import { resourceKey, useClientMutation } from '../../shared/api/hooks';
import { getSavings, withdraw } from './api';
export const useSavings = () => useQuery({ queryKey: resourceKey('savings'), queryFn: ({ signal }) => getSavings(signal), refetchInterval: 15000 });
export const useWithdraw = () => useClientMutation(withdraw);

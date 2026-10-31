import { useQuery } from '@tanstack/react-query';
import { getSavings, withdraw, deposit } from './api';
import { resourceKey, useClientMutation } from '../../shared/api/hooks';
export const useSavings = () => useQuery({ queryKey: resourceKey('savings'), queryFn: ({ signal }) => getSavings(signal), refetchInterval: 15000 });
export const useWithdraw = () => useClientMutation(withdraw);
export const useDeposit = () => useClientMutation(deposit);
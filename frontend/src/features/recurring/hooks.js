import { useQuery, useQueryClient } from '@tanstack/react-query';
import { resourceKey, useClientMutation } from '../../shared/api/hooks';
import { getRecurring, setExclusion } from './api';
export const useRecurring = () => useQuery({ queryKey: resourceKey('recurring'), queryFn: ({ signal }) => getRecurring(signal), select: (data) => data || [] });
export function useSetExclusion() {
  const cache = useQueryClient();
  return useClientMutation(setExclusion, { onSuccess: (payments) => cache.setQueryData(resourceKey('recurring'), payments) });
}

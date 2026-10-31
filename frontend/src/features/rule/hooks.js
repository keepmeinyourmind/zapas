import { useQuery, useQueryClient } from '@tanstack/react-query';
import { resourceKey, useClientMutation } from '../../shared/api/hooks';
import { putRule, patchRule } from './api';

// В контракте нет GET /rule. Храним в памяти только подтверждённые ответы PUT/PATCH.
export const useKnownRule = () => useQuery({ queryKey: resourceKey('rule'), enabled: false, gcTime: Infinity });
export function useSaveRule(partial = false) {
  const cache = useQueryClient();
  return useClientMutation(partial ? patchRule : putRule, {
    onSuccess: (rule) => cache.setQueryData(resourceKey('rule'), rule),
  });
}

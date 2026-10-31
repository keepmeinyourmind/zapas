import { useMutation, useQueryClient } from '@tanstack/react-query';
import { config } from '../../app/config';

export const clientKey = ['client', config.clientId];
export const resourceKey = (name, params) => params === undefined ? [...clientKey, name] : [...clientKey, name, params];

export function useClientMutation(mutationFn, options = {}) {
  const cache = useQueryClient();
  return useMutation({
    mutationFn,
    ...options,
    onSettled: async (data, error, variables) => {
      await options.onSettled?.(data, error, variables);
      // Ответ 409 может означать, что состояние изменилось из-за другого запроса, таймаута или события банка.
      await cache.invalidateQueries({ queryKey: clientKey, predicate: (q) => q.queryKey[2] !== 'rule' });
    },
  });
}

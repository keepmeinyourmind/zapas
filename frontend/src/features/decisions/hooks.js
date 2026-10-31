import { useQuery, useQueryClient } from '@tanstack/react-query';
import { resourceKey, useClientMutation } from '../../shared/api/hooks';
import { getDecisions, answerDecision } from './api';
export const useDecisions = (params = {}, { enabled = true } = {}) => useQuery({ queryKey: resourceKey('decisions', params), queryFn: ({ signal }) => getDecisions(params, signal), select: (data) => data || [], refetchInterval: 15000, enabled });
export function useAnswerDecision() {
  const cache = useQueryClient();
  return useClientMutation(answerDecision, {
    onSuccess: (result) => {
      // Ответ сервера сразу заменяет старый вопрос, не дожидаясь повторных GET.
      cache.setQueriesData({ queryKey: resourceKey('decisions') }, (items) => {
        if (!items) return items;
        return items.map((item) => item.id === result.id ? result : item);
      });
    },
  });
}

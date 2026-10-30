import { useQuery } from '@tanstack/react-query';
import { resourceKey, useClientMutation } from '../../shared/api/hooks';
import { getDecisions, answerDecision } from './api';
export const useDecisions = (params = {}, { enabled = true } = {}) => useQuery({ queryKey: resourceKey('decisions', params), queryFn: ({ signal }) => getDecisions(params, signal), select: (data) => data || [], refetchInterval: 15000, enabled });
export const useAnswerDecision = () => useClientMutation(answerDecision);

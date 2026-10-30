import { useQuery, useQueryClient } from '@tanstack/react-query';
import { resourceKey, useClientMutation } from '../../shared/api/hooks';
import { submitOnboarding } from './api';
import { readOnboardingReceipt, recordOnboarding } from './status';

export const useOnboardingStatus = () => useQuery({
  queryKey: resourceKey('onboarding'),
  enabled: false,
  initialData: readOnboardingReceipt,
  gcTime: Infinity,
});
export function useSubmitOnboarding() {
  const cache = useQueryClient();
  return useClientMutation(submitOnboarding, {
    onSuccess: async (result) => {
      recordOnboarding(cache, result);
      await cache.resetQueries({ queryKey: resourceKey('rule'), exact: true });
    },
  });
}

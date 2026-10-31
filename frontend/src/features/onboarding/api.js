import { request, clientPath } from '../../shared/api/client';
export const submitOnboarding = (body) => request(`${clientPath}/onboarding`, { method: 'POST', body });

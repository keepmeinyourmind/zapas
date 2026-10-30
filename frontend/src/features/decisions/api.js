import { request, clientPath } from '../../shared/api/client';
export function getDecisions({ status, limit = 50 } = {}, signal) {
  const params = new URLSearchParams({ limit });
  if (status) params.set('status', status);
  return request(`${clientPath}/decisions?${params}`, { signal });
}
export const answerDecision = ({ id, ...body }) => request(`/api/decisions/${encodeURIComponent(id)}/answer`, { method: 'POST', body });

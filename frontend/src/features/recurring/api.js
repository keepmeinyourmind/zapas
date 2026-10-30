import { request, clientPath } from '../../shared/api/client';
export const getRecurring = (signal) => request(`${clientPath}/recurring`, { signal });
export const setExclusion = (body) => request(`${clientPath}/recurring/exclusion`, { method: 'PUT', body });

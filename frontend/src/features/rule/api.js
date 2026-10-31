import { request, clientPath } from '../../shared/api/client';
export const putRule = (body) => request(`${clientPath}/rule`, { method: 'PUT', body });
export const patchRule = (body) => request(`${clientPath}/rule`, { method: 'PATCH', body });

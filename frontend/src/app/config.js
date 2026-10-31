export const config = {
  apiBaseUrl: (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, ''),
  clientId: import.meta.env.VITE_CLIENT_ID || 'anya',
  demoTools: import.meta.env.VITE_ENABLE_DEMO_TOOLS === 'true',
};

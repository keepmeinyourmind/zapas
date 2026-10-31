import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10000,
      retry: (count, error) => error.status >= 500 && count < 1,
    },
    mutations: { retry: false },
  },
});
const ToastContext = createContext(null);
export const useToast = () => useContext(ToastContext);

export default function Providers({ children }) {
  const [message, setMessage] = useState('');
  const timer = useRef();
  const toast = useCallback((text) => {
    clearTimeout(timer.current);
    setMessage(text);
    timer.current = setTimeout(() => setMessage(''), 3500);
  }, []);
  useEffect(() => () => clearTimeout(timer.current), []);
  return <QueryClientProvider client={queryClient}><ToastContext.Provider value={toast}>
    {children}<div className={`toast${message ? ' visible' : ''}`} role="status" aria-live="polite">{message}</div>
  </ToastContext.Provider></QueryClientProvider>;
}

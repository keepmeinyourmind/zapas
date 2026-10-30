import { useQuery } from '@tanstack/react-query';
import { checkHealth } from './api';
import { config } from '../../app/config';
import AppButton from '../../shared/ui/AppButton';
import { InlineError } from '../../shared/ui/States';

export default function ServiceHealth() {
  const query = useQuery({ queryKey: ['system', 'health', config.apiBaseUrl], queryFn: ({ signal }) => checkHealth(signal), retry: false });
  const available = query.isSuccess && query.data?.status === 'ok';
  return <article className="card"><div className="card-body p-4">
    <h2 className="section-title mb-3">Подключение к API</h2>
    <p className={`small mb-2${available ? ' text-success' : ' text-secondary'}`} role="status">{query.isFetching ? 'Проверяем соединение…' : available ? 'API работает' : 'API недоступен'}</p>
    <p className="form-text mb-3">Тестовый клиент: {config.clientId}</p>
    <AppButton onClick={() => query.refetch()} pending={query.isFetching}>Проверить соединение</AppButton>
    <InlineError error={query.error} />
  </div></article>;
}

import AppButton from './AppButton';
export function LoadingState() {
  return <div className="state-message" role="status"><span className="spinner-border spinner-border-sm me-2" aria-hidden="true" />Загружаем данные…</div>;
}
export function EmptyState({ children = 'Здесь пока ничего нет.' }) {
  return <p className="state-message mb-0">{children}</p>;
}
export function ErrorState({ error, retry }) {
  return <div className="state-message" role="alert"><p className="mb-2">{error?.message || 'Не удалось загрузить данные.'}</p>{retry && <AppButton onClick={retry}>Повторить</AppButton>}</div>;
}
export function InlineError({ error }) {
  return error ? <p className="form-error mt-3 mb-0" role="alert">{error.message || error}</p> : null;
}

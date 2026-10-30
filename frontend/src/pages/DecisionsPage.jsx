import { useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { useDecisions } from '../features/decisions/hooks';
import DecisionCard from '../features/decisions/DecisionCard';
import { LoadingState, ErrorState, EmptyState } from '../shared/ui/States';
import AppButton from '../shared/ui/AppButton';
export default function DecisionsPage() {
  const { savings, onExplain, onRule } = useOutletContext();
  const [filter, setFilter] = useState('all');
  const [limit, setLimit] = useState(50);
  // К открытым вопросам относятся и записи в очереди; статуса "done" в API нет.
  const query = useDecisions({ limit });
  const items = query.data?.filter((decision) => filter === 'all' || (['pending', 'queued'].includes(decision.status) ? filter === 'pending' : filter === 'done')) || [];
  return <section className="view" id="view-decisions"><div className="mb-4"><h1 className="page-title">Решения</h1><p className="page-description">История действий системы и причины</p></div>
    <div className="d-flex flex-wrap gap-2 mb-3 filter-row" role="group" aria-label="Фильтр решений">{[['all', 'Все'], ['pending', 'Открытые вопросы'], ['done', 'Завершённые']].map(([value, text]) => <button className={`btn filter-chip${filter === value ? ' active' : ''}`} type="button" key={value} aria-pressed={filter === value} onClick={() => setFilter(value)}>{text}</button>)}</div>
    <article className="card"><div className="card-body p-3 p-lg-4">{query.isPending ? <LoadingState /> : query.isError ? <ErrorState error={query.error} retry={query.refetch} /> : <>
      {items.length ? items.map((decision) => <DecisionCard key={decision.id} decision={decision} blocked={!savings.data || savings.isError || savings.data.arrested || savings.data.paused} onExplain={onExplain} onRule={onRule} />) : <EmptyState>Решений в этой категории пока нет.</EmptyState>}
      {query.data.length >= limit && (limit < 200 ? <AppButton className="mt-3" onClick={() => setLimit(Math.min(200, limit + 50))}>Показать больше</AppButton> : <p className="form-text mt-3 mb-0">Показаны последние 200 решений.</p>)}
    </>}</div></article>
  </section>;
}

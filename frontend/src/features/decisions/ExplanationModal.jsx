import Modal from 'react-bootstrap/Modal';
import { formatMoney } from '../../shared/utils/money';
import { formatDate } from '../../shared/utils/date';
export default function ExplanationModal({ decision, onHide }) {
  const explanation = decision?.explanation;
  return <Modal show={Boolean(decision)} onHide={onHide} centered aria-labelledby="explanation-title">
    <Modal.Header closeButton closeLabel="Закрыть"><Modal.Title as="h2" id="explanation-title">Как рассчитана безопасная сумма</Modal.Title></Modal.Header>
    <Modal.Body>{explanation ? <>
      <p className="text-secondary small">Система оставляет деньги на предстоящие расходы и резерв, а откладывает только безопасный остаток.</p>
      <div className="list-group list-group-flush">
        <div className="list-group-item d-flex justify-content-between gap-3"><span>Собственные средства на карте</span><strong className="text-nowrap">{formatMoney(explanation.card_balance)}</strong></div>
        {(explanation.upcoming || []).map((item, i) => <div className="list-group-item d-flex justify-content-between gap-3" key={`${item.title}-${i}`}><span>{item.title}{item.at && <small className="d-block text-secondary">{formatDate(item.at)}</small>}{item.per_day != null && <small className="d-block text-secondary">{formatMoney(item.per_day)} в день × {item.days} дн.</small>}</span><strong className="text-nowrap">− {formatMoney(item.amount)}</strong></div>)}
        <div className="list-group-item d-flex justify-content-between gap-3"><span>Всего предстоящих расходов</span><strong className="text-nowrap">− {formatMoney(explanation.upcoming_total)}</strong></div>
        <div className="list-group-item d-flex justify-content-between gap-3"><span>Неприкосновенный остаток</span><strong className="text-nowrap">− {formatMoney(explanation.buffer)}</strong></div>
        <div className="list-group-item d-flex justify-content-between gap-3"><strong>Безопасно отложить</strong><strong className="text-success text-nowrap">{formatMoney(explanation.safe_amount)}</strong></div>
      </div>
      <p className="form-text mt-3 mb-0">Расчёт от {formatDate(decision.created_at, true)} · до следующего дохода {formatDate(explanation.until)}. Текущий остаток может измениться.</p>
      {explanation.summary && <details className="form-advanced mt-3"><summary>Пояснение</summary><p className="text-secondary small summary-text mt-2 mb-0">{explanation.summary}</p></details>}
    </> : <p className="text-secondary small mb-0">Для этого решения расчёт не требуется.</p>}</Modal.Body>
  </Modal>;
}

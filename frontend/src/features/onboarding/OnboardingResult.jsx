import AppButton from '../../shared/ui/AppButton';
export default function OnboardingResult({ result, hasSavings, onRule, onEdit }) {
  if (!result) return null;
  return <div className="status-notice mt-3" role="status">
    {result.eligible ? <>
      <strong>{hasSavings ? 'Анкета обновлена' : 'Копилка вам доступна'}</strong>
      <p className="mt-2 mb-0">{result.mode === 'simple' ? 'Простой режим: фиксированный процент от поступлений.' : 'Умный режим: с анализом доходов и расходов.'}</p>
      {!hasSavings && <><p className="mt-2 mb-0">Анкета одобрена. Следующий шаг — цель и правило накопления.</p><AppButton className="mt-3" onClick={onRule}>Настроить правило</AppButton>{onEdit && <button className="section-link mt-3" type="button" onClick={onEdit}>Заполнить анкету заново</button>}</>}
    </> : <><strong>Сейчас подключить копилку не получится</strong>{result.reasons?.length > 0 && <ul className="mt-2 mb-0">{result.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul>}{hasSavings && <p className="mt-2 mb-0">Накопление приостановлено. Накопленные деньги остаются в копилке.</p>}</>}
  </div>;
}

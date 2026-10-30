import { moneyNumber, formatMoney } from '../../shared/utils/money';
import Icon from '../../shared/ui/Icon';
import AppButton from '../../shared/ui/AppButton';
import { readWithdrawal } from '../../shared/utils/withdrawalAttempt';
import { config } from '../../app/config';

export default function SavingsCard({ savings, overview = false, rule, onRule, onWithdraw }) {
  const progress = Math.min(100, Math.max(0, savings.progress_percent || 0));
  const state = savings.arrested ? 'Ограничения' : savings.paused ? 'На паузе' : 'Активна';
  const hasUnconfirmedWithdrawal = Boolean(readWithdrawal(config.clientId));
  const canWithdraw = hasUnconfirmedWithdrawal || (savings.balance > 0 && !savings.arrested);
  return <article className={`card savings-card${overview ? ' savings-overview' : ''}`}><div className="card-body p-4">
    <div className={overview ? 'savings-overview-top d-flex align-items-start justify-content-between gap-2' : ''}>
      <div className="savings-overview-heading-copy"><div className="savings-label">Моя копилка</div><div className="savings-name">{savings.goal_name}</div></div>
      {overview && <span className={`savings-state${savings.paused || savings.arrested ? ' paused' : ''}`}>{state}</span>}
    </div>
    <div className={overview ? 'savings-overview-center' : ''}>
      <div className="savings-amount">{moneyNumber(savings.balance)} <small>₽</small></div>
      <div className="savings-caption">Накоплено</div>
      {hasUnconfirmedWithdrawal && <p className="form-text mt-2 mb-0">Снятие ожидает подтверждения. Откройте «Снять», чтобы проверить результат.</p>}
      <div className="d-flex justify-content-between gap-2 goal-row mt-4"><span>Прогресс к цели</span><span><b>{Math.round(progress)}</b>% · {formatMoney(savings.goal_amount)}</span></div>
      <div className="progress" role="progressbar" aria-label="Прогресс накоплений" aria-valuenow={progress} aria-valuemin={0} aria-valuemax={100}><div className="progress-bar" style={{ width: `${progress}%` }} /></div>
      {overview ? <div className="savings-actions d-flex justify-content-center gap-3">
        <button className="savings-action" type="button" onClick={onRule}><span className="savings-action-icon"><Icon name="plus" /></span><span>Правило</span></button>
        <button className="savings-action" type="button" onClick={onWithdraw} disabled={!canWithdraw}><span className="savings-action-icon"><Icon name="withdraw" /></span><span>Снять</span></button>
      </div> : <div className="mt-3"><div className="savings-rule">{rule ? <>Правило: <b>{rule.save_percent}</b>% от дохода</> : 'Процент накопления указан в вашем правиле'}</div><AppButton className="mt-3" onClick={onWithdraw} disabled={!canWithdraw}>Снять на карту</AppButton></div>}
    </div>
  </div></article>;
}

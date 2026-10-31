import Icon from './Icon';
export default function InfoButton({ onClick, label = 'Как рассчитана безопасная сумма', disabled }) {
  return <button className="stat-info" type="button" onClick={onClick} disabled={disabled} aria-label={label} title={label}><Icon name="info" /></button>;
}

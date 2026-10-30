import { Link } from 'react-router-dom';
import AppButton from '../../shared/ui/AppButton';

export default function ConnectionNotice({ onboarding, onRule }) {
  const approved = onboarding?.eligible === true;
  return <div className="status-notice">
    <strong>{approved ? 'Осталось настроить правило' : 'Подключение копилки'}</strong>
    <p className="mt-2 mb-3">{approved
      ? 'Анкета одобрена. Укажите цель и параметры накопления, чтобы подключить копилку.'
      : 'Сначала заполните анкету. Если копилка вам доступна, следующим шагом настроим цель и правило накопления.'}</p>
    {approved ? <AppButton onClick={onRule}>Настроить правило</AppButton>
      : <Link className="reference-button text-decoration-none" to="/onboarding">Заполнить анкету</Link>}
  </div>;
}

import { useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import OnboardingForm from '../features/onboarding/OnboardingForm';
import OnboardingResult from '../features/onboarding/OnboardingResult';
export default function OnboardingPage() {
  const { savings, onboarding, onRule } = useOutletContext();
  const [editing, setEditing] = useState(false);
  const approved = onboarding?.eligible === true;
  const showForm = Boolean(savings.data) || !approved || editing;
  return <section className="view" id="view-onboarding"><div className="mb-4"><h1 className="page-title">{savings.data ? 'Анкета копилки' : 'Подключение копилки'}</h1><p className="page-description">{savings.data ? 'Информация о доходах и обязательных платежах' : approved && !editing ? 'Шаг 2 из 2 · Цель и правило' : 'Шаг 1 из 2 · Анкета'}</p></div>
    {savings.data && <p className="connection-note mb-3">Новая анкета заменит прежнюю. Если накопление окажется недоступно, копилка встанет на паузу.</p>}
    {showForm && <article className="card onboarding-card"><div className="card-body p-3 p-lg-4"><OnboardingForm hasSavings={Boolean(savings.data)} onChecked={() => setEditing(false)} /></div></article>}
    <OnboardingResult result={editing && approved ? null : onboarding} hasSavings={Boolean(savings.data)} onRule={onRule} onEdit={() => setEditing(true)} />
  </section>;
}

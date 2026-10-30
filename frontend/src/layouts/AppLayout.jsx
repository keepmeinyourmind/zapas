import { useEffect, useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { config } from '../app/config';
import { useSavings } from '../features/savings/hooks';
import RuleModal from '../features/rule/RuleModal';
import WithdrawModal from '../features/savings/WithdrawModal';
import ExplanationModal from '../features/decisions/ExplanationModal';
import Icon from '../shared/ui/Icon';
import { useOnboardingStatus } from '../features/onboarding/hooks';
import ConnectionNotice from '../features/onboarding/ConnectionNotice';
import { LoadingState, ErrorState } from '../shared/ui/States';

export const navigation = [
  { path: '/', key: 'home', label: 'Главная' },
  { path: '/savings', key: 'savings', label: 'Копилка и правило' },
  { path: '/transactions', key: 'transactions', label: 'Операции' },
  { path: '/recurring', key: 'recurring', label: 'Регулярные платежи' },
  { path: '/decisions', key: 'decisions', label: 'Решения' },
  { path: '/onboarding', key: 'onboarding', label: 'Редактирование' },
];
export function Navigation({ connected = true }) {
  return <nav className="menu-wrap" aria-label="Разделы кабинета"><div className="card menu-panel"><div className="row g-1">
    {navigation.map((item) => <div className="col-6" key={item.key}><Link className={`menu-option w-100${item.key === 'home' ? ' active' : ''}`} to={item.path} aria-current={item.key === 'home' ? 'page' : undefined}><span className="menu-option-icon"><Icon name={item.key} /></span><span className="menu-option-label">{item.key === 'onboarding' && !connected ? 'Подключение копилки' : item.label}</span></Link></div>)}
  </div></div></nav>;
}
export default function AppLayout() {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const savings = useSavings();
  const { data: onboarding } = useOnboardingStatus();
  const [modal, setModal] = useState(null);
  const [decision, setDecision] = useState(null);
  const home = pathname === '/';
  const missing = !savings.data && savings.error?.status === 404;
  const canConfigureRule = onboarding?.eligible !== false && (
    !savings.isError && Boolean(savings.data) || missing && onboarding?.eligible === true
  );
  const onRule = () => {
    if (canConfigureRule) setModal('rule');
    else navigate('/onboarding');
  };
  const context = { savings, onboarding, onRule, onWithdraw: () => setModal('withdraw'), onExplain: setDecision };
  const needsSavings = ['/transactions', '/recurring', '/decisions'].includes(pathname);
  const pageLabel = pathname === '/onboarding' ? savings.data ? 'Редактирование' : 'Подключение копилки' : navigation.find((item) => item.path === pathname)?.label || 'Запас';
  useEffect(() => {
    window.scrollTo({ top: 0 });
    setModal(null);
    setDecision(null);
  }, [pathname]);
  return <div className="main-area">
    <header className="topbar d-flex align-items-center justify-content-between px-3 px-lg-5"><Link className="brand text-decoration-none" to="/">Запас</Link><div className="crumb">Личный кабинет <span className="mx-1">/</span><strong>{pageLabel}</strong></div></header>
    <main className="content-area container-fluid py-4 px-3 px-lg-5"><div className="content-wrap mx-auto">
      {!home && <div id="back-to-home" className="mb-3"><Link className="back-button text-decoration-none" to="/" aria-label="Назад на главную"><Icon name="back" /><span>Назад</span></Link></div>}
      {savings.data?.arrested && <div className="home-column status-notice mb-3" role="status">На счёт наложены ограничения. Пополнения, снятие и ответы на вопросы временно недоступны.</div>}
      {needsSavings && !savings.data ? <section className="view" id="view-connection"><h1 className="page-title mb-4">{pageLabel}</h1>{savings.isPending ? <LoadingState /> : missing ? <ConnectionNotice onboarding={onboarding} onRule={onRule} /> : <ErrorState error={savings.error} retry={savings.refetch} />}</section> : <Outlet context={context} />}
      {config.demoTools && <div className="home-column mt-4"><Link className="section-link" to="/demo">Проверка копилки</Link></div>}
    </div></main>
    <RuleModal show={modal === 'rule' && canConfigureRule} onHide={() => setModal(null)} savings={savings.data} onSaved={({ created }) => { setModal(null); if (created) navigate('/'); }} onRequiresOnboarding={() => { setModal(null); navigate('/onboarding'); }} />
    <WithdrawModal show={modal === 'withdraw'} onHide={() => setModal(null)} savings={savings.data} />
    <ExplanationModal decision={decision} onHide={() => setDecision(null)} />
  </div>;
}

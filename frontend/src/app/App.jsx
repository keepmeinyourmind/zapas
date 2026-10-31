import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { config } from './config';
import AppLayout from '../layouts/AppLayout';
import DashboardPage from '../pages/DashboardPage';
import SavingsPage from '../pages/SavingsPage';
import TransactionsPage from '../pages/TransactionsPage';
import RecurringPage from '../pages/RecurringPage';
import DecisionsPage from '../pages/DecisionsPage';
import OnboardingPage from '../pages/OnboardingPage';
import DemoPage from '../pages/DemoPage';

export default function App() {
  return <BrowserRouter><Routes><Route element={<AppLayout />}>
    <Route index element={<DashboardPage />} />
    <Route path="savings" element={<SavingsPage />} />
    <Route path="transactions" element={<TransactionsPage />} />
    <Route path="recurring" element={<RecurringPage />} />
    <Route path="decisions" element={<DecisionsPage />} />
    <Route path="onboarding" element={<OnboardingPage />} />
    {config.demoTools && <Route path="demo" element={<DemoPage />} />}
    <Route path="*" element={<Navigate to="/" replace />} />
  </Route></Routes></BrowserRouter>;
}

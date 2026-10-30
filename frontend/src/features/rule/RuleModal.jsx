import Modal from 'react-bootstrap/Modal';
import RuleForm from './RuleForm';

export default function RuleModal({ show, onHide, savings, onSaved, onRequiresOnboarding }) {
  return <Modal show={show} onHide={onHide} centered aria-labelledby="rule-title">
    <Modal.Header closeButton closeLabel="Закрыть"><Modal.Title as="h2" id="rule-title">Цель и правило</Modal.Title></Modal.Header>
    {show && <RuleForm savings={savings} onSaved={onSaved} onRequiresOnboarding={onRequiresOnboarding} />}
  </Modal>;
}

import { useState } from 'react';
import { Modal, Button, Form } from 'react-bootstrap';
import { useDeposit } from './hooks';

export default function DepositModal({ show, onHide, onSuccess }) {
  const [amount, setAmount] = useState('');
  const depositMutation = useDeposit();

  const handleSubmit = async (e) => {
    e.preventDefault();
    const amountInKopecks = parseInt(amount) * 100;
    
    try {
      await depositMutation.mutateAsync({ amount: amountInKopecks });
      setAmount('');
      onSuccess?.();
      onHide();
    } catch (error) {
      alert(`Ошибка: ${error.message}`);
    }
  };

  return (
    <Modal show={show} onHide={onHide}>
      <Modal.Header closeButton>
        <Modal.Title>Пополнить копилку</Modal.Title>
      </Modal.Header>
      <Form onSubmit={handleSubmit}>
        <Modal.Body>
          <Form.Group>
            <Form.Label>Сумма (₽)</Form.Label>
            <Form.Control
              type="number"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="Введите сумму"
              min="1"
              required
            />
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={onHide}>
            Отмена
          </Button>
          <Button variant="primary" type="submit" disabled={depositMutation.isLoading}>
            {depositMutation.isLoading ? 'Пополнение...' : 'Пополнить'}
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
}
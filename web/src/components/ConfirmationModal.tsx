import { AlertTriangle } from 'lucide-react';

import React from 'react';

import { Button } from './ui/Button';
import { Modal } from './ui/Modal';

interface ConfirmationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  isDangerous?: boolean;
}

const ConfirmationModal: React.FC<ConfirmationModalProps> = ({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  isDangerous = false,
}) => {
  return (
    <Modal isOpen={isOpen} onClose={onClose} hideCloseButton={true}>
      <div style={{ padding: '20px', maxWidth: '400px' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '15px',
            marginBottom: '15px',
          }}
        >
          {isDangerous && (
            <div
              style={{
                backgroundColor: 'rgba(239, 68, 68, 0.1)',
                padding: '10px',
                borderRadius: '50%',
                color: '#ef4444',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <AlertTriangle size={24} />
            </div>
          )}
          <h2 style={{ margin: 0, fontSize: '1.25rem' }}>{title}</h2>
        </div>

        <p
          style={{
            marginBottom: '25px',
            lineHeight: '1.5',
            color: 'var(--text-muted)',
          }}
        >
          {message}
        </p>

        <div
          style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}
        >
          <Button variant="secondary" onClick={onClose}>
            {cancelText}
          </Button>
          <Button
            variant={isDangerous ? 'danger' : 'primary'}
            onClick={() => {
              onConfirm();
              onClose();
            }}
          >
            {confirmText}
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ConfirmationModal;

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
    <Modal isOpen={isOpen} onClose={onClose} title={title}>
      <div style={{ padding: '20px', maxWidth: '400px' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'start',
            gap: '15px',
            marginBottom: '25px',
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
                flexShrink: 0,
              }}
            >
              <AlertTriangle size={24} />
            </div>
          )}
          <p
            style={{
              margin: 0,
              lineHeight: '1.5',
              color: 'var(--text-muted)',
            }}
          >
            {message}
          </p>
        </div>

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

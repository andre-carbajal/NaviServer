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
      <div className="tw:max-w-[400px] tw:p-5">
        <div className="tw:mb-[25px] tw:flex tw:items-start tw:gap-[15px]">
          {isDangerous && (
            <div className="tw:flex tw:shrink-0 tw:items-center tw:justify-center tw:rounded-full tw:bg-red-500/10 tw:p-2.5 tw:text-red-500">
              <AlertTriangle size={24} />
            </div>
          )}
          <p className="tw:m-0 tw:leading-[1.5] tw:text-text-muted">
            {message}
          </p>
        </div>

        <div className="tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2 tw:max-[769px]:[&_button]:w-full">
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

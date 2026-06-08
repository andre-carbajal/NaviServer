import { X } from 'lucide-react';

import React from 'react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
  title: string;
  hideCloseButton?: boolean;
  contentClassName?: string;
}

export const Modal: React.FC<ModalProps> = ({
  isOpen,
  onClose,
  children,
  title,
  hideCloseButton,
  contentClassName,
}) => {
  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className={`modal-content ${contentClassName || ''}`.trim()}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">{title}</h2>
          {!hideCloseButton && (
            <button className="icon-action" onClick={onClose} type="button">
              <X size={20} />
            </button>
          )}
        </div>
        {children}
      </div>
    </div>
  );
};

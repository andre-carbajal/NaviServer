import { X } from 'lucide-react';

import React, { useId } from 'react';

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
  const titleId = useId();

  if (!isOpen) return null;

  return (
    <dialog
      open
      className="modal-overlay"
      aria-labelledby={titleId}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      <div className={`modal-content ${contentClassName || ''}`.trim()}>
        <div className="modal-header">
          <h2 id={titleId} className="modal-title">
            {title}
          </h2>
          {!hideCloseButton && (
            <button className="icon-action" onClick={onClose} type="button">
              <X size={20} />
            </button>
          )}
        </div>
        {children}
      </div>
    </dialog>
  );
};

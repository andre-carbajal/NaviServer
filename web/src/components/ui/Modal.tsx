import { X } from 'lucide-react';

import React, { useId } from 'react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
  title: React.ReactNode;
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
      className="tw:fixed tw:inset-0 tw:z-[1000] tw:m-0 tw:flex tw:h-screen tw:max-h-none tw:w-screen tw:max-w-none tw:items-center tw:justify-center tw:border-0 tw:bg-black/70 tw:p-0 tw:text-inherit tw:backdrop:bg-transparent"
      aria-labelledby={titleId}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      <div
        className={`tw:box-border tw:max-h-[calc(100vh-40px)] tw:w-full tw:max-w-[500px] tw:overflow-y-auto tw:rounded-xl tw:border tw:border-border tw:bg-bg-card tw:p-[30px] tw:shadow-[0_4px_20px_rgba(0,0,0,0.5)] tw:max-[769px]:max-w-[calc(100%-40px)]! tw:max-[769px]:mx-5 tw:max-[769px]:p-5 tw:max-[481px]:mx-4 tw:max-[481px]:p-4 ${contentClassName || ''}`.trim()}
      >
        <div className="tw:mb-5 tw:flex tw:flex-row tw:items-center tw:justify-between tw:gap-3">
          <h2 id={titleId} className="tw:m-0 tw:text-[1.2rem] tw:font-bold">
            {title}
          </h2>
          {!hideCloseButton && (
            <button
              className="tw:flex tw:h-9 tw:w-9 tw:cursor-pointer tw:items-center tw:justify-center tw:rounded-sm tw:border-0 tw:bg-transparent tw:p-0 tw:text-text-muted tw:transition-all tw:duration-200 tw:ease-[ease] tw:hover:bg-white/10 tw:hover:text-white"
              onClick={onClose}
              type="button"
            >
              <X size={20} className="tw:h-[18px] tw:w-[18px]" />
            </button>
          )}
        </div>
        {children}
      </div>
    </dialog>
  );
};

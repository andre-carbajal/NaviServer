import { useCallback, useRef, useState } from 'react';

import { Button } from '../components/ui/Button';
import { Modal } from '../components/ui/Modal';

type DialogVariant = 'default' | 'danger' | 'success';

type AlertOptions = {
  title: string;
  message: string;
  confirmText?: string;
  variant?: DialogVariant;
};

type ConfirmOptions = AlertOptions & {
  cancelText?: string;
};

type DialogState =
  ({ type: 'alert' } & AlertOptions) | ({ type: 'confirm' } & ConfirmOptions);

const getButtonVariant = (variant?: DialogVariant) => {
  if (variant === 'danger') return 'danger';
  return 'primary';
};

export const useModalDialog = () => {
  const [dialogState, setDialogState] = useState<DialogState | null>(null);
  const resolverRef = useRef<((result: boolean) => void) | null>(null);

  const closeDialog = useCallback((result: boolean) => {
    resolverRef.current?.(result);
    resolverRef.current = null;
    setDialogState(null);
  }, []);

  const showAlert = useCallback((options: AlertOptions) => {
    resolverRef.current?.(false);
    return new Promise<void>((resolve) => {
      resolverRef.current = () => resolve();
      setDialogState({ type: 'alert', ...options });
    });
  }, []);

  const showConfirm = useCallback((options: ConfirmOptions) => {
    resolverRef.current?.(false);
    return new Promise<boolean>((resolve) => {
      resolverRef.current = resolve;
      setDialogState({ type: 'confirm', ...options });
    });
  }, []);

  const modalDialog = dialogState ? (
    <Modal
      isOpen
      onClose={() => closeDialog(false)}
      title={dialogState.title}
      contentClassName="tw:max-w-[460px]!"
    >
      <div className="tw:w-full tw:max-w-[420px] tw:mx-auto">
        <p
          className={`tw:m-0 tw:leading-[1.5] ${dialogState.variant === 'danger' ? 'tw:rounded-lg tw:border tw:border-danger/30 tw:bg-danger/10 tw:p-3 tw:text-red-200' : 'tw:text-text-muted'}`}
        >
          {dialogState.message}
        </p>
        <div className="tw:mt-[25px] tw:flex tw:justify-end tw:gap-2.5 tw:max-[769px]:flex-col tw:max-[769px]:gap-2">
          {dialogState.type === 'confirm' && (
            <Button variant="secondary" onClick={() => closeDialog(false)}>
              {dialogState.cancelText || 'Cancel'}
            </Button>
          )}
          <Button
            variant={getButtonVariant(dialogState.variant)}
            onClick={() => closeDialog(true)}
          >
            {dialogState.confirmText || 'OK'}
          </Button>
        </div>
      </div>
    </Modal>
  ) : null;

  return { showAlert, showConfirm, modalDialog };
};

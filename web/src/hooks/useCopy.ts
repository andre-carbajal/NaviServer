import { useCallback, useState } from 'react';

export const useCopy = (timeout: number = 2000) => {
  const [copied, setCopied] = useState(false);

  const fallbackCopy = useCallback((text: string) => {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.focus();
    ta.select();
    try {
      document.execCommand('copy');
      document.body.removeChild(ta);
      return true;
    } catch (e) {
      document.body.removeChild(ta);
      console.error('Fallback copy failed', e);
      return false;
    }
  }, []);

  const copy = useCallback(
    async (text: string) => {
      let success: boolean;
      try {
        if (navigator.clipboard && navigator.clipboard.writeText) {
          await navigator.clipboard.writeText(text);
          success = true;
        } else {
          success = fallbackCopy(text);
        }
      } catch (err) {
        console.error('Copy failed', err);
        success = fallbackCopy(text);
      }

      if (success) {
        setCopied(true);
        setTimeout(() => setCopied(false), timeout);
      }
      return success;
    },
    [fallbackCopy, timeout],
  );

  return { copied, copy };
};

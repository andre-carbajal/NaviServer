import axios from 'axios';

export const getApiErrorMessage = (
  error: unknown,
  fallback: string,
): string => {
  if (!axios.isAxiosError(error)) return fallback;

  const data: unknown = error.response?.data;
  if (typeof data === 'string' && data.trim()) {
    return data.trim();
  }

  if (data && typeof data === 'object') {
    const record = data as Record<string, unknown>;
    for (const key of ['message', 'error']) {
      const value = record[key];
      if (typeof value === 'string' && value.trim()) {
        return value.trim();
      }
    }
  }

  return error.message || fallback;
};

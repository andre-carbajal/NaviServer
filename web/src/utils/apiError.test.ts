import axios from 'axios';
import { describe, expect, it } from 'vitest';

import { getApiErrorMessage } from './apiError';

describe('getApiErrorMessage', () => {
  it('uses a plain text response body', () => {
    const error = new axios.AxiosError('Request failed', 'ERR_BAD_REQUEST');
    error.response = {
      data: 'server must be stopped before deletion\n',
      status: 409,
      statusText: 'Conflict',
      headers: new axios.AxiosHeaders(),
      config: { headers: new axios.AxiosHeaders() },
    };

    expect(getApiErrorMessage(error, 'Fallback')).toBe(
      'server must be stopped before deletion',
    );
  });

  it('uses message or error fields from JSON responses', () => {
    const messageError = new axios.AxiosError('Request failed');
    messageError.response = {
      data: { message: 'Unable to start server' },
      status: 400,
      statusText: 'Bad Request',
      headers: new axios.AxiosHeaders(),
      config: { headers: new axios.AxiosHeaders() },
    };
    const errorField = new axios.AxiosError('Request failed');
    errorField.response = {
      data: { error: 'Permission denied' },
      status: 403,
      statusText: 'Forbidden',
      headers: new axios.AxiosHeaders(),
      config: { headers: new axios.AxiosHeaders() },
    };

    expect(getApiErrorMessage(messageError, 'Fallback')).toBe(
      'Unable to start server',
    );
    expect(getApiErrorMessage(errorField, 'Fallback')).toBe(
      'Permission denied',
    );
  });

  it('falls back for non-Axios errors', () => {
    expect(getApiErrorMessage(new Error('Unexpected'), 'Fallback')).toBe(
      'Fallback',
    );
  });
});

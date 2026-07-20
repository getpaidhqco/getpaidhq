import { AxiosError } from 'axios';
import { HTTPError } from '../types';

export class GetPaidHQError extends Error {
  public readonly statusCode?: number;
  public readonly errorCode?: string;
  public readonly details?: Record<string, any>;
  public readonly requestId?: string;

  constructor(
    message: string,
    statusCode?: number,
    errorCode?: string,
    details?: Record<string, any>,
    requestId?: string,
  ) {
    super(message);
    this.name = 'GetPaidHQError';
    this.statusCode = statusCode;
    this.errorCode = errorCode;
    this.details = details;
    this.requestId = requestId;
    Object.setPrototypeOf(this, GetPaidHQError.prototype);
  }
}

export class AuthenticationError extends GetPaidHQError {
  constructor(message = 'Authentication failed', details?: Record<string, any>) {
    super(message, 401, 'authentication_error', details);
    this.name = 'AuthenticationError';
    Object.setPrototypeOf(this, AuthenticationError.prototype);
  }
}

export class AuthorizationError extends GetPaidHQError {
  constructor(message = 'Authorization failed', details?: Record<string, any>) {
    super(message, 403, 'authorization_error', details);
    this.name = 'AuthorizationError';
    Object.setPrototypeOf(this, AuthorizationError.prototype);
  }
}

export class NotFoundError extends GetPaidHQError {
  constructor(resource: string, id?: string) {
    const message = id ? `${resource} with id '${id}' not found` : `${resource} not found`;
    super(message, 404, 'not_found', { resource, id });
    this.name = 'NotFoundError';
    Object.setPrototypeOf(this, NotFoundError.prototype);
  }
}

export class ValidationError extends GetPaidHQError {
  constructor(message: string, details?: Record<string, any>) {
    super(message, 400, 'validation_error', details);
    this.name = 'ValidationError';
    Object.setPrototypeOf(this, ValidationError.prototype);
  }
}

export class RateLimitError extends GetPaidHQError {
  public readonly retryAfter?: number;

  constructor(message = 'Rate limit exceeded', retryAfter?: number) {
    super(message, 429, 'rate_limit_error', { retryAfter });
    this.name = 'RateLimitError';
    this.retryAfter = retryAfter;
    Object.setPrototypeOf(this, RateLimitError.prototype);
  }
}

export class ServerError extends GetPaidHQError {
  constructor(message = 'Internal server error', statusCode = 500) {
    super(message, statusCode, 'server_error');
    this.name = 'ServerError';
    Object.setPrototypeOf(this, ServerError.prototype);
  }
}

export function handleApiError(error: AxiosError<HTTPError>): never {
  const response = error.response;
  const requestId = response?.headers['x-request-id'] as string | undefined;

  if (!response) {
    throw new GetPaidHQError(
      error.message || 'Network error occurred',
      undefined,
      'network_error',
      undefined,
      requestId,
    );
  }

  const { status } = response;
  const data = response.data as
    | (Partial<HTTPError> & { error?: string; message?: string; details?: Record<string, any> })
    | undefined;
  const message = data?.title || data?.detail || data?.error || data?.message || error.message;
  const details = (data?.errors as unknown as Record<string, any>) ?? data?.details;

  switch (status) {
    case 400:
      throw new ValidationError(message, details);
    case 401:
      throw new AuthenticationError(message, details);
    case 403:
      throw new AuthorizationError(message, details);
    case 404:
      throw new NotFoundError('Resource', undefined);
    case 429: {
      const retryAfter = response.headers['retry-after'];
      throw new RateLimitError(message, retryAfter ? parseInt(retryAfter as string) : undefined);
    }
    case 500:
    case 502:
    case 503:
    case 504:
      throw new ServerError(message, status);
    default:
      throw new GetPaidHQError(message, status, 'api_error', details, requestId);
  }
}

export type ApplicationErrorCode =
  | 'conflict'
  | 'unauthorized'
  | 'forbidden'
  | 'not-found'
  | 'transient'
  | 'invalid';

export class ApplicationError extends Error {
  constructor(public readonly code: ApplicationErrorCode, message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = 'ApplicationError';
  }
}

export function errorFromStatus(status: number): ApplicationErrorCode {
  if (status === 409) return 'conflict';
  if (status === 401) return 'unauthorized';
  if (status === 403) return 'forbidden';
  if (status === 404) return 'not-found';
  if (status >= 400 && status < 500) return 'invalid';
  return 'transient';
}

export function toApplicationError(error: unknown): ApplicationError {
  if (error instanceof ApplicationError) return error;
  return new ApplicationError('transient', error instanceof Error ? error.message : 'Unexpected application error', {
    cause: error,
  });
}

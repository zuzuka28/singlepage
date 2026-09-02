import { describe, expect, it } from 'vitest';
import { passwordValidationError } from './password';

describe('password validation', () => {
  it('accepts passwords with at least eight characters, a letter, and a number', () => {
    expect(passwordValidationError('secure12')).toBeNull();
    expect(passwordValidationError('пароль12')).toBeNull();
  });

  it('rejects passwords shorter than eight characters', () => {
    expect(passwordValidationError('pass123')).toBe('Use at least 8 characters.');
  });

  it('rejects passwords without a letter or a number', () => {
    expect(passwordValidationError('password')).toBe('Use at least one letter and one number.');
    expect(passwordValidationError('12345678')).toBe('Use at least one letter and one number.');
  });
});

export const minimumPasswordLength = 8;
export const passwordPattern = String.raw`(?=.*\p{L})(?=.*\p{Nd}).{${minimumPasswordLength},}`;

export function passwordValidationError(password: string): string | null {
  if (password.length < minimumPasswordLength) {
    return `Use at least ${minimumPasswordLength} characters.`;
  }
  if (!/\p{L}/u.test(password) || !/\p{Nd}/u.test(password)) {
    return 'Use at least one letter and one number.';
  }
  return null;
}

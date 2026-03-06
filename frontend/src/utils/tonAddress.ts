import { Address } from '@ton/core';

/**
 * Converts a TON address (raw 0:xxx or friendly) to human-readable base64url format (UQ...).
 */
export function toFriendlyAddress(rawOrFriendly: string): string {
  try {
    const addr = Address.parse(rawOrFriendly);
    return addr.toString({ urlSafe: true, bounceable: false });
  } catch {
    return rawOrFriendly;
  }
}

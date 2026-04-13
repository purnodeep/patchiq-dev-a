import { describe, it, expect } from 'vitest';
import { formatDeploymentId } from '../../lib/format';

describe('formatDeploymentId', () => {
  it('formats a normal UUID to D-XXXXXX uppercase with 6 chars', () => {
    const result = formatDeploymentId('a1b2c3d4-e5f6-7890-abcd-ef1234567890');
    expect(result).toBe('D-A1B2C3');
  });

  it('handles short input of 3 chars without error', () => {
    expect(formatDeploymentId('abc')).toBe('D-ABC');
  });

  it('handles empty string without error', () => {
    expect(formatDeploymentId('')).toBe('D-');
  });

  it('keeps already uppercase input uppercase', () => {
    expect(formatDeploymentId('ABCDEF123')).toBe('D-ABCDEF');
  });
});

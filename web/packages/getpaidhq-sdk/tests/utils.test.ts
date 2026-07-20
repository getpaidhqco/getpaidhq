import { describe, it, expect } from 'vitest';

// Simple utility function to test
function sum(a: number, b: number): number {
  return a + b;
}

describe('Utils', () => {
  it('should correctly sum two numbers', () => {
    expect(sum(2, 3)).toBe(5);
    expect(sum(-1, 1)).toBe(0);
    expect(sum(0, 0)).toBe(0);
  });
});

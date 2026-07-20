import { describe, expect, it } from 'vitest';
import { GetPaidHQClient } from '../src';

describe('GetPaidHQ SDK', () => {
  it('should initialize with API key', () => {
    const client = new GetPaidHQClient({
      apiKey: 'sk_test_123',
    });

    expect(client).toBeDefined();
    expect(client.customers).toBeDefined();
    expect(client.coupons).toBeDefined();
    expect(client.discounts).toBeDefined();
    expect(client.subscriptions).toBeDefined();
    expect(client.products).toBeDefined();
  });

  it('should initialize with bearer token getToken', () => {
    const client = new GetPaidHQClient({
      getToken: async () => 'test_bearer_token',
    });

    expect(client).toBeDefined();
  });
  it('should initialize with token and append', async () => {
    const client = new GetPaidHQClient({
      baseURL: 'http://localhost:8081',
      token: 'test_bearer_token',
    });

    // Test the AuthManager directly
    const authManager = (client as any).authManager;
    const testConfig = { headers: {}, url: '/test' };
    const authResult = await authManager.applyAuth(testConfig);

    console.log('Auth result:', JSON.stringify(authResult, null, 2));
    console.log('Auth type:', authManager.getAuthType());

    // Verify the token was added as a query parameter
    expect(authResult.params).toBeDefined();
    expect(authResult.params.token).toBe('test_bearer_token');
    expect(client).toBeDefined();
  });

  it('should throw error without authentication', () => {
    expect(() => {
      new GetPaidHQClient({} as any);
    }).toThrow('Either apiKey, bearerToken, or token must be provided');
  });

  it('should allow updating authentication', () => {
    const client = new GetPaidHQClient({
      apiKey: 'sk_test_123',
    });

    // Should not throw
    client.updateApiKey('sk_new_key');
    client.updateBearerToken('new_token');
  });
});

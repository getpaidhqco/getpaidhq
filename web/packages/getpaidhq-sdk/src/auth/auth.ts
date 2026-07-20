import { AxiosRequestConfig } from 'axios';

export interface AuthConfig {
  apiKey?: string;
  getToken?: () => Promise<string | null>;
  token?: string;
}

export class AuthManager {
  private config: AuthConfig;

  constructor(config: AuthConfig) {
    if (!config.apiKey && !config.getToken && !config.token) {
      throw new Error('Either apiKey, bearerToken, or token must be provided');
    }
    this.config = config;
  }

  async applyAuth(requestConfig: AxiosRequestConfig): Promise<AxiosRequestConfig> {
    if (!requestConfig.headers) {
      requestConfig.headers = {};
    }

    // API Key takes precedence over Bearer token, token is used as query parameter
    if (this.config.apiKey) {
      requestConfig.headers['X-API-Key'] = this.config.apiKey;
    } else if (this.config.getToken) {
      requestConfig.headers['Authorization'] = `Bearer ${await this.config.getToken()}`;
    } else if (this.config.token) {
      // Add token as query parameter for public endpoints
      // Ensure params object exists and merge with any existing params
      // Merge our token with any existing params
      requestConfig.params = {
        ...requestConfig.params,
        token: this.config.token,
      };
    }

    return requestConfig;
  }

  updateApiKey(apiKey: string): void {
    this.config.apiKey = apiKey;
  }

  getAuthType(): 'apiKey' | 'bearer' | 'token' | null {
    if (this.config.apiKey) return 'apiKey';
    if (this.config.getToken) return 'bearer';
    if (this.config.token) return 'token';
    return null;
  }
}

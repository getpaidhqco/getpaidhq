"use client"

import React, {useState} from 'react';
import {LoginComponentProps} from "@getpaidhq/auth-core/types";

export const ApiKeyLoginComponent: React.FC<LoginComponentProps> = (props) => {
  const [apiKey, setApiKey] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!apiKey) {
      setError('API Key is required');
      return;
    }

    // Here you would typically validate the API key and store it
    // For demo purposes, we'll just log it
    console.log('API Key submitted:', apiKey);
    localStorage.setItem('apiKey', apiKey);
    window.location.href = '/dashboard';
  };

  return (
    <div className="flex min-h-svh flex-col items-center justify-center bg-muted p-6 md:p-10">
      <div className="w-full max-w-sm md:max-w-3xl">
        <div className="rounded-lg border bg-card text-card-foreground shadow-sm p-6">
          <h2 className="text-2xl font-semibold mb-4">API Key Login</h2>

          <form onSubmit={handleSubmit}>
            <div className="mb-4">
              <label htmlFor="apiKey" className="block text-sm font-medium mb-1">
                API Key
              </label>
              <input
                id="apiKey"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                className="w-full p-2 border rounded-md"
                placeholder="Enter your API key"
              />
              {error && <p className="text-red-500 text-sm mt-1">{error}</p>}
            </div>

            <button
              type="submit"
              className="w-full bg-primary text-primary-foreground hover:bg-primary/90 py-2 px-4 rounded-md"
            >
              Sign In
            </button>
          </form>
        </div>
      </div>
    </div>
  );
};

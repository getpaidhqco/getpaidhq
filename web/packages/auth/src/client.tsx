"use client"

import React from 'react';
import {LoginComponentProps, OrgSwitcherComponentProps, UserProfileComponentProps} from '@getpaidhq/auth-core/types';
import {ClerkLoginComponent} from '@getpaidhq/auth-clerk/client'
import {ApiKeyLoginComponent} from '@getpaidhq/auth-apikey/client'
import {ClerkOrgSwitcher} from "@getpaidhq/auth-clerk/org-switcher";
import {ClerkUserProfile} from "@getpaidhq/auth-clerk/user-profile";

export function LoginComponent(props: LoginComponentProps) {
  const providerName = process.env.NEXT_PUBLIC_AUTH_PROVIDER ?? 'apiKey';
  switch (providerName) {
    case 'clerk':
      return <ClerkLoginComponent {...props} />;
    case 'apiKey':
      return <ApiKeyLoginComponent {...props} />;
    default:
      throw new Error(`Unknown auth provider: ${providerName}`);
  }
}

export function OrgSwitcherComponent(props: OrgSwitcherComponentProps) {
  const providerName = process.env.NEXT_PUBLIC_AUTH_PROVIDER ?? 'apiKey';
  switch (providerName) {
    case 'clerk':
      return <ClerkOrgSwitcher {...props} />;
    case 'apiKey':
      return <ClerkOrgSwitcher {...props} />;
    default:
      throw new Error(`Unknown auth provider: ${providerName}`);
  }
}

export function UserProfileComponent(props: UserProfileComponentProps) {
  const providerName = process.env.NEXT_PUBLIC_AUTH_PROVIDER ?? 'apiKey';
  switch (providerName) {
    case 'clerk':
      return <ClerkUserProfile {...props} />;
    case 'apiKey':
      // For apiKey, we'll use the same component as clerk for now
      return <ClerkUserProfile {...props} />;
    default:
      throw new Error(`Unknown auth provider: ${providerName}`);
  }
}
"use client"

import React from 'react';
import { UserProfile, SignedIn } from "@clerk/nextjs";
import { UserProfileComponentProps } from "@getpaidhq/auth-core/types";

export const ClerkUserProfile: React.FC<UserProfileComponentProps> = (props) => {
  return (
    <div className="w-full">
      <SignedIn>
        <UserProfile

          appearance={props.appearance}
        />
      </SignedIn>
    </div>
  );
};

"use client"

import React from 'react';
import {SignIn} from "@clerk/nextjs";
import {LoginComponentProps} from "@getpaidhq/auth-core/types";

export const ClerkLoginComponent: React.FC<LoginComponentProps> = (props) => {
  return (
    <div className="flex min-h-svh flex-col items-center justify-center bg-muted p-6 md:p-10">
      <div className="w-full max-w-sm md:max-w-3xl">
        <SignIn {...props} />
      </div>
    </div>
  );
};



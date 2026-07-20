"use client"

import React from 'react';
import {OrganizationSwitcher, SignedIn} from "@clerk/nextjs";
import {OrgSwitcherComponentProps} from "@getpaidhq/auth-core/types";

export const ClerkOrgSwitcher: React.FC<OrgSwitcherComponentProps> = (props) => {
  return (
    <div className="p-2 w-full">
      <SignedIn>
        <OrganizationSwitcher
          hidePersonal={true}
          afterCreateOrganizationUrl={props.afterChangeUrl}
          afterSelectOrganizationUrl={props.afterChangeUrl}
        />
      </SignedIn>
    </div>
  );
};



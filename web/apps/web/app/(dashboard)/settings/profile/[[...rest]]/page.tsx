"use client"

import { useAuth } from "@getpaidhq/auth"
import { UserProfileComponent } from "@getpaidhq/auth/client"

import { SettingsProvider } from "@/app/(dashboard)/settings/settings-context"

export default function ProfileSettings() {
  const { orgId } = useAuth()

  return (
    <SettingsProvider parentId={orgId} id="profile">
      <UserProfileComponent
        appearance={{
          variables: {
            borderRadius: 0,
            colorShadow: "transparent",
            borderColor: "transparent",
          },
        }}
      />
    </SettingsProvider>
  )
}

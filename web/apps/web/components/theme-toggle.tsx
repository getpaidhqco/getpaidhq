"use client"

import * as React from "react"
import { Moon, Sun } from "lucide-react"
import { useTheme } from "@/components/theme-provider"

import { Button } from "@/components/ui/button"

export function ModeToggle() {
  const { theme, resolvedTheme, setTheme } = useTheme()
  const [mounted, setMounted] = React.useState(false)
  React.useEffect(() => setMounted(true), [])

  if (!mounted) {
    return (
      <Button variant="outline" size="icon-sm" aria-label="Toggle dark mode">
        <Sun className="size-3.5" />
      </Button>
    )
  }

  const current = theme === "system" ? resolvedTheme : theme
  const next = current === "dark" ? "light" : "dark"

  return (
    <Button
      variant="outline"
      size="icon-sm"
      aria-label={`Switch to ${next} mode`}
      onClick={() => setTheme(next)}
    >
      {current === "dark" ? (
        <Moon className="size-3.5" />
      ) : (
        <Sun className="size-3.5" />
      )}
    </Button>
  )
}

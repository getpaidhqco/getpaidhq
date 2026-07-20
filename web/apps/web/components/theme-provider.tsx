"use client"

import * as React from "react"

type Theme = "light" | "dark" | "system"
type Resolved = "light" | "dark"

interface ThemeContextValue {
  theme: Theme
  resolvedTheme: Resolved
  setTheme: (theme: Theme) => void
}

const ThemeContext = React.createContext<ThemeContextValue | null>(null)

const STORAGE_KEY = "theme"

function readStoredTheme(): Theme {
  if (typeof window === "undefined") return "system"
  try {
    const v = window.localStorage.getItem(STORAGE_KEY)
    if (v === "light" || v === "dark" || v === "system") return v
  } catch {
    // localStorage can throw in iframes/private mode — ignore
  }
  return "system"
}

function systemPreference(): Resolved {
  if (typeof window === "undefined") return "light"
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
}

function applyClass(resolved: Resolved) {
  if (typeof document === "undefined") return
  document.documentElement.classList.toggle("dark", resolved === "dark")
}

export function ThemeProvider({
  children,
  defaultTheme = "system",
}: {
  children: React.ReactNode
  defaultTheme?: Theme
}) {
  const [theme, setThemeState] = React.useState<Theme>(defaultTheme)
  const [resolvedTheme, setResolvedTheme] = React.useState<Resolved>("light")

  // Initial sync — read storage, compute resolved, apply class.
  React.useEffect(() => {
    const stored = readStoredTheme()
    setThemeState(stored)
    const resolved = stored === "system" ? systemPreference() : stored
    setResolvedTheme(resolved)
    applyClass(resolved)
  }, [])

  // Track system changes when theme is "system".
  React.useEffect(() => {
    if (theme !== "system" || typeof window === "undefined") return
    const mq = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = () => {
      const r: Resolved = mq.matches ? "dark" : "light"
      setResolvedTheme(r)
      applyClass(r)
    }
    mq.addEventListener("change", handler)
    return () => mq.removeEventListener("change", handler)
  }, [theme])

  const setTheme = React.useCallback((next: Theme) => {
    try {
      window.localStorage.setItem(STORAGE_KEY, next)
    } catch {
      // ignore
    }
    setThemeState(next)
    const resolved = next === "system" ? systemPreference() : next
    setResolvedTheme(resolved)
    applyClass(resolved)
  }, [])

  const value = React.useMemo<ThemeContextValue>(
    () => ({ theme, resolvedTheme, setTheme }),
    [theme, resolvedTheme, setTheme],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const ctx = React.useContext(ThemeContext)
  if (ctx) return ctx
  // Safe fallback if a consumer renders outside the provider.
  return {
    theme: "system",
    resolvedTheme: "light",
    setTheme: () => {},
  }
}

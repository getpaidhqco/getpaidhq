"use client"

import * as React from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  Bell,
  Boxes,
  CreditCard,
  FileText,
  Gauge,
  HelpCircle,
  LayoutDashboard,
  LogOut,
  Menu,
  Plus,
  Receipt,
  RefreshCw,
  Search,
  Settings,
  ShoppingBag,
  Sparkles,
  Tag,
  Users,
} from "lucide-react"

import { useAuth } from "@getpaidhq/auth"
import { OrgSwitcherComponent } from "@getpaidhq/auth/client"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet"
import { ModeToggle } from "@/components/theme-toggle"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type NavItem = {
  href: string
  label: string
  icon: React.ComponentType<{ className?: string }>
}

type NavGroup = {
  section?: string
  items: NavItem[]
}

const NAV: NavGroup[] = [
  {
    items: [{ href: "/dashboard", label: "Dashboard", icon: LayoutDashboard }],
  },
  {
    section: "Catalog",
    items: [
      { href: "/products", label: "Products", icon: Boxes },
      { href: "/meters", label: "Usage meters", icon: Gauge },
      { href: "/discounts", label: "Discounts", icon: Tag },
    ],
  },
  {
    section: "Revenue",
    items: [
      { href: "/subscriptions", label: "Subscriptions", icon: RefreshCw },
      { href: "/invoices", label: "Invoices", icon: FileText },
      { href: "/orders", label: "Orders", icon: ShoppingBag },
      { href: "/payments", label: "Payments", icon: CreditCard },
    ],
  },
  {
    section: "Audience",
    items: [{ href: "/customers", label: "Customers", icon: Users }],
  },
]

function SidebarBody({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname()
  const { currentUser, logout } = useAuth()

  const initials = React.useMemo(() => {
    const name = currentUser?.name ?? currentUser?.email ?? ""
    return name
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((s: string) => s[0]?.toUpperCase() ?? "")
      .join("") || "?"
  }, [currentUser])

  return (
    <div className="flex h-full flex-col bg-sidebar text-sidebar-foreground">
      <div className="flex h-14 items-center gap-2 border-b border-sidebar-border px-4">
        <span className="grid size-7 place-items-center rounded-md bg-foreground text-background">
          <svg viewBox="0 0 16 16" className="size-4" fill="currentColor">
            <path d="M3 8a5 5 0 0 1 10 0 .75.75 0 0 1-1.5 0 3.5 3.5 0 1 0-3.5 3.5h2.25a.75.75 0 0 1 0 1.5H8A5 5 0 0 1 3 8Zm5 0a.75.75 0 0 1 .75-.75h3a.75.75 0 0 1 0 1.5h-3A.75.75 0 0 1 8 8Z" />
          </svg>
        </span>
        <span className="text-sm font-semibold tracking-tight">GetPaidHQ</span>
        <span className="ml-auto rounded-sm border border-sidebar-border bg-background px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
          Live
        </span>
      </div>

      <div className="px-3 pt-3 pb-1">
        <OrgSwitcherComponent />
      </div>

      <nav className="flex-1 overflow-y-auto px-3 pb-3">
        {NAV.map((group, gi) => (
          <div key={gi} className="mb-1">
            {group.section ? (
              <div className="mt-3 mb-1 px-2 font-mono text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {group.section}
              </div>
            ) : null}
            <ul role="list" className="flex flex-col gap-0.5">
              {group.items.map((item) => {
                const Icon = item.icon
                const active =
                  pathname === item.href ||
                  (item.href !== "/dashboard" && pathname?.startsWith(item.href))
                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      onClick={onNavigate}
                      aria-current={active ? "page" : undefined}
                      className={cn(
                        "group flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm transition",
                        active
                          ? "bg-sidebar-accent text-sidebar-accent-foreground"
                          : "text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                      )}
                    >
                      <Icon className="size-4" />
                      <span className="truncate">{item.label}</span>
                    </Link>
                  </li>
                )
              })}
            </ul>
          </div>
        ))}

        <div className="mt-3 border-t border-sidebar-border pt-3">
          <Link
            href="/settings"
            onClick={onNavigate}
            className="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <Settings className="size-4" />
            Settings
          </Link>
          <Link
            href="/design-system"
            onClick={onNavigate}
            className="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <Sparkles className="size-4" />
            Design system
          </Link>
        </div>
      </nav>

      <div className="border-t border-sidebar-border p-3">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              className="mb-2 flex w-full items-center gap-2.5 rounded-md px-1.5 py-1 text-left transition hover:bg-sidebar-accent"
            >
              <span className="grid size-7 shrink-0 place-items-center rounded-md bg-brand-soft text-brand text-[10px] font-semibold">
                {initials}
              </span>
              <div className="min-w-0 flex-1 text-[11px] leading-tight">
                <div className="truncate font-medium">
                  {currentUser?.name ?? currentUser?.email ?? "User"}
                </div>
                <div className="truncate text-muted-foreground">
                  {currentUser?.email}
                </div>
              </div>
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuItem onSelect={() => logout()}>
              <LogOut className="size-4" />
              Sign out
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link href="/settings">
                <Settings className="size-4" />
                Settings
              </Link>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <div className="flex items-center justify-end gap-2">
          <ModeToggle />
        </div>
      </div>
    </div>
  )
}

function Topbar({
  title,
  breadcrumbs,
}: {
  title?: string
  breadcrumbs?: { href: string; label: string }[]
}) {
  const [mobileOpen, setMobileOpen] = React.useState(false)

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-border bg-background/85 px-4 backdrop-blur lg:px-6">
      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetTrigger asChild>
          <Button
            variant="ghost"
            size="icon-sm"
            className="lg:hidden"
            aria-label="Open menu"
          >
            <Menu className="size-5" />
          </Button>
        </SheetTrigger>
        <SheetContent
          side="left"
          className="w-72 border-r border-sidebar-border p-0"
        >
          <SidebarBody onNavigate={() => setMobileOpen(false)} />
        </SheetContent>
      </Sheet>

      <div className="hidden min-w-0 items-baseline gap-2 lg:flex">
        {breadcrumbs?.length ? (
          <ol
            role="list"
            className="flex items-baseline gap-1.5 font-mono text-[11px] uppercase tracking-wider text-muted-foreground"
          >
            {breadcrumbs.map((b, i) => (
              <li key={b.href} className="flex items-baseline gap-1.5">
                <Link href={b.href} className="hover:text-foreground">
                  {b.label}
                </Link>
                {i < breadcrumbs.length - 1 ? <span aria-hidden>/</span> : null}
              </li>
            ))}
          </ol>
        ) : null}
        {title ? <span className="truncate text-sm font-medium">{title}</span> : null}
      </div>

      <div className="ml-auto flex w-full max-w-md items-center">
        <div className="relative w-full">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="search"
            aria-label="Search anything"
            placeholder="Search customers, invoices, payments…"
            className="block h-8 w-full rounded-md border border-border bg-muted/40 pr-12 pl-8 text-sm text-foreground placeholder:text-muted-foreground focus:border-foreground/30 focus:bg-background focus:outline-none"
          />
          <kbd className="absolute top-1/2 right-2 hidden -translate-y-1/2 rounded border border-border bg-background px-1 py-0.5 font-mono text-[10px] text-muted-foreground sm:inline-flex">
            ⌘K
          </kbd>
        </div>
      </div>

      <div className="hidden items-center gap-1 md:flex">
        <span className="inline-flex h-7 items-center gap-1.5 rounded-md border border-warning/30 bg-warning/10 px-2 font-mono text-[10px] uppercase tracking-wider text-warning">
          <span className="size-1.5 rounded-full bg-warning" />
          Test mode
        </span>
        <Button variant="ghost" size="icon-sm" aria-label="Notifications">
          <Bell className="size-4" />
        </Button>
        <Button variant="ghost" size="icon-sm" aria-label="Help">
          <HelpCircle className="size-4" />
        </Button>
        <Separator orientation="vertical" className="mx-1 h-5" />
        <Button size="sm">
          <Plus className="size-3.5" data-icon="inline-start" />
          Create
        </Button>
      </div>
    </header>
  )
}

export function AppShell({
  children,
  title,
  breadcrumbs,
}: {
  children: React.ReactNode
  title?: string
  breadcrumbs?: { href: string; label: string }[]
}) {
  return (
    <div className="flex min-h-dvh">
      <aside className="hidden w-60 shrink-0 border-r border-sidebar-border lg:block">
        <div className="sticky top-0 h-dvh">
          <SidebarBody />
        </div>
      </aside>
      <div className="min-w-0 flex-1">
        <Topbar title={title} breadcrumbs={breadcrumbs} />
        <main className="px-4 py-6 lg:px-8 lg:py-8">{children}</main>
      </div>
    </div>
  )
}

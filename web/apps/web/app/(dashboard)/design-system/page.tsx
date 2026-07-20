"use client";

import * as React from "react";
import Link from "next/link";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from "recharts";
import {
  Bell,
  CalendarDays,
  Check,
  CreditCard,
  Download,
  Search,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable, type Column } from "@/components/ui/data-table";
import { Input } from "@/components/ui/input";
import { KpiRow } from "@/components/ui/kpi-row";
import { Label } from "@/components/ui/label";
import { PageHeader } from "@/components/ui/page-header";
import { SectionHead } from "@/components/ui/section-head";
import { Separator } from "@/components/ui/separator";
import { StatusTag, type StatusTone } from "@/components/ui/status-tag";
import { Surface } from "@/components/ui/surface";
import { Switch } from "@/components/ui/switch";

const MRR_SERIES = [
  { month: "Jun", mrr: 62100, expansion: 4200 },
  { month: "Jul", mrr: 64580, expansion: 4900 },
  { month: "Aug", mrr: 66920, expansion: 5300 },
  { month: "Sep", mrr: 69540, expansion: 5800 },
  { month: "Oct", mrr: 71880, expansion: 6100 },
  { month: "Nov", mrr: 73420, expansion: 6400 },
  { month: "Dec", mrr: 75150, expansion: 6900 },
  { month: "Jan", mrr: 76420, expansion: 7100 },
  { month: "Feb", mrr: 77100, expansion: 7400 },
  { month: "Mar", mrr: 77860, expansion: 7800 },
  { month: "Apr", mrr: 78200, expansion: 8000 },
  { month: "May", mrr: 78420, expansion: 8300 },
];

const TABLE_SAMPLE = [
  { id: "pay_8x2nqkpw", customer: "Acme Industries", amount: "$2,800", status: "succeeded" as const },
  { id: "pay_91kfp22a", customer: "Globex Corp", amount: "$480", status: "pending" as const },
  { id: "pay_77ax9zqe", customer: "Initech", amount: "$1,200", status: "failed" as const },
  { id: "pay_33ckp1xx", customer: "Hooli", amount: "$240", status: "refunded" as const },
];

type Tx = (typeof TABLE_SAMPLE)[number];
const TX_STATUS: Record<Tx["status"], { l: string; t: StatusTone }> = {
  succeeded: { l: "Succeeded", t: "success" },
  pending: { l: "Pending", t: "info" },
  failed: { l: "Failed", t: "danger" },
  refunded: { l: "Refunded", t: "neutral" },
};

const TX_COLS: Column<Tx>[] = [
  {
    key: "id",
    header: "Reference",
    render: (r) => <span className="font-mono text-xs text-foreground">{r.id}</span>,
  },
  {
    key: "customer",
    header: "Customer",
    render: (r) => <span className="text-sm font-medium text-foreground">{r.customer}</span>,
  },
  {
    key: "amount",
    header: "Amount",
    align: "right",
    render: (r) => <span className="tabular font-medium">{r.amount}</span>,
  },
  {
    key: "status",
    header: "Status",
    render: (r) => (
      <StatusTag tone={TX_STATUS[r.status].t}>{TX_STATUS[r.status].l}</StatusTag>
    ),
  },
];

const TOKEN_GROUPS: {
  title: string;
  hint: string;
  tokens: { name: string; var: string }[];
}[] = [
  {
    title: "Canvas",
    hint: "Page background, surface, and text.",
    tokens: [
      { name: "background", var: "--background" },
      { name: "foreground", var: "--foreground" },
      { name: "card", var: "--card" },
      { name: "muted", var: "--muted" },
      { name: "muted-foreground", var: "--muted-foreground" },
    ],
  },
  {
    title: "Brand",
    hint: "One saturated accent. Used sparingly for primary actions, focus rings, and chart-1.",
    tokens: [
      { name: "brand", var: "--brand" },
      { name: "brand-soft", var: "--brand-soft" },
      { name: "primary", var: "--primary" },
      { name: "border", var: "--border" },
      { name: "border-strong", var: "--border-strong" },
    ],
  },
  {
    title: "Status",
    hint: "Coordinated tones tuned to the brand temperature — not stock Tailwind.",
    tokens: [
      { name: "success", var: "--success" },
      { name: "info", var: "--info" },
      { name: "warning", var: "--warning" },
      { name: "destructive", var: "--destructive" },
    ],
  },
  {
    title: "Chart",
    hint: "Monochromatic: brand + warm complement + slate scale. Never rainbow.",
    tokens: [
      { name: "chart-1 · brand", var: "--chart-1" },
      { name: "chart-2 · complement", var: "--chart-2" },
      { name: "chart-3", var: "--chart-3" },
      { name: "chart-4", var: "--chart-4" },
      { name: "chart-5", var: "--chart-5" },
    ],
  },
];

const KPI_DEMO = [
  { label: "Monthly recurring", value: "$78,420", delta: "+5.8%", sub: "vs last 30d" },
  { label: "Active customers", value: "412", delta: "+12", sub: "+12 net new" },
  { label: "Net retention", value: "108%", delta: "+2.1pp", sub: "rolling 90d" },
  { label: "Failed rate", value: "3.2%", delta: "-0.4pp", sub: "recovering 71%" },
];

const SPACING = [
  { name: "1", px: 4 },
  { name: "2", px: 8 },
  { name: "3", px: 12 },
  { name: "4", px: 16 },
  { name: "5", px: 20 },
  { name: "6", px: 24 },
  { name: "8", px: 32 },
  { name: "10", px: 40 },
  { name: "12", px: 48 },
  { name: "16", px: 64 },
];

export default function DesignSystemPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-14">
      <PageHeader
        eyebrow="Precision direction · v1.0"
        title="A neutral, data-dense system for subscription billing"
        description={
          <>
            Every token, font, surface, and component is driven by CSS variables in{" "}
            <code className="font-mono text-foreground">app/globals.css</code>. Toggle dark mode in
            the sidebar footer — this whole page updates live.
          </>
        }
      />

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Principles"
          title="What makes this not look AI-generated"
          subtitle="The rules we use to avoid the default Tailwind + shadcn 'every-section-is-a-card' template look."
        />
        <ol className="grid grid-cols-1 gap-x-10 gap-y-6 md:grid-cols-2">
          {[
            {
              n: "01",
              t: "Whitespace before borders",
              d: "Default to typographic hierarchy and space. Only add a border when content actually needs containment — not as decoration.",
            },
            {
              n: "02",
              t: "Cards only when interactive",
              d: "Reserve bordered Surface cards for things that act on their own (a clickable subscription, a navigable tile). Sibling content sits directly on the canvas.",
            },
            {
              n: "03",
              t: "Tinted neutrals, never pure",
              d: "Pure #fff and pure neutral-grey have no temperature personality. Precision tints its neutrals with a hint of cool slate-blue.",
            },
            {
              n: "04",
              t: "One brand color, used sparingly",
              d: "A single saturated deep-teal accent. Most of the page is neutral; the brand picks out a focus ring, a primary action, chart-1.",
            },
            {
              n: "05",
              t: "Monochromatic charts",
              d: "Brand + a single warm complement + slate-scale grays. Never a 5-hue rainbow imported from Tailwind defaults.",
            },
            {
              n: "06",
              t: "Mono is a label, not a body font",
              d: "Geist Mono is reserved for IDs, timestamps, status tags, and eyebrows — never for paragraph text.",
            },
          ].map((row) => (
            <li key={row.n} className="border-t border-border pt-4">
              <div className="flex items-baseline gap-3">
                <span className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                  {row.n}
                </span>
                <span className="text-sm font-semibold text-foreground">{row.t}</span>
              </div>
              <p className="mt-1.5 text-pretty text-sm text-muted-foreground">{row.d}</p>
            </li>
          ))}
        </ol>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Tokens"
          title="Color palette"
          subtitle="Live values from the active theme. All driven by CSS variables — change one block in globals.css to remix."
        />
        <div className="grid grid-cols-1 gap-x-10 gap-y-8 md:grid-cols-2">
          {TOKEN_GROUPS.map((g) => (
            <div key={g.title} className="flex flex-col gap-2">
              <div>
                <div className="text-sm font-semibold text-foreground">{g.title}</div>
                <p className="text-xs text-muted-foreground">{g.hint}</p>
              </div>
              <ul role="list">
                {g.tokens.map((t, i) => (
                  <li
                    key={t.var}
                    className={
                      i !== g.tokens.length - 1
                        ? "flex items-center gap-3 border-b border-border py-2"
                        : "flex items-center gap-3 py-2"
                    }
                  >
                    <span
                      className="size-5 shrink-0 rounded-sm border border-border"
                      style={{ background: `var(${t.var})` }}
                    />
                    <span className="flex-1 text-sm text-foreground">{t.name}</span>
                    <code className="font-mono text-[11px] text-muted-foreground">{t.var}</code>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Tokens"
          title="Type scale"
          subtitle="Inter (with cv02/03/04/11) for everything · Geist Mono for IDs, timestamps and eyebrows"
        />
        <ul role="list">
          {[
            {
              meta: "Display · 48",
              cls: "text-5xl font-semibold tracking-tight",
              sample: "Subscription billing, simplified.",
            },
            { meta: "H1 · 24", cls: "text-2xl font-semibold tracking-tight", sample: "Dashboard" },
            { meta: "H2 · 16", cls: "text-base font-semibold", sample: "Revenue by product" },
            {
              meta: "Body · 14",
              cls: "text-sm text-foreground",
              sample:
                "All recurring revenue agreements across customers, products, and currencies.",
            },
            {
              meta: "Meta · 12",
              cls: "text-xs text-muted-foreground",
              sample: "Renews 1 Jun 2026 · ZAR 4,200 / mo",
            },
            { meta: "Eyebrow · 10", cls: "eyebrow", sample: "Trailing 30 days" },
            {
              meta: "Mono · 12",
              cls: "font-mono text-xs text-muted-foreground",
              sample: "cus_8a3kf2 · INV-002841 · pay_8x2nqkpw",
            },
            { meta: "num-display · 40", cls: "num-display text-4xl tabular", sample: "$78,420.00" },
          ].map((row, i, arr) => (
            <li
              key={row.meta}
              className={
                i !== arr.length - 1
                  ? "grid grid-cols-[140px_1fr] items-baseline gap-4 border-b border-border py-3"
                  : "grid grid-cols-[140px_1fr] items-baseline gap-4 py-3"
              }
            >
              <div className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                {row.meta}
              </div>
              <div className={row.cls}>{row.sample}</div>
            </li>
          ))}
        </ul>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Patterns"
          title="Surfaces · when to box, when not to"
          subtitle="The single biggest call in this system. Default is no box."
        />
        <div className="grid grid-cols-1 gap-x-10 gap-y-6 md:grid-cols-2">
          <div>
            <div className="flex items-baseline justify-between border-b border-border pb-2">
              <h3 className="text-sm font-semibold">Borderless section (default)</h3>
              <span className="font-mono text-[10px] uppercase tracking-wider text-success">
                Use 90% of the time
              </span>
            </div>
            <div className="py-4">
              <p className="eyebrow">MRR</p>
              <div className="num-display mt-1 text-3xl tabular text-foreground">$78,420</div>
              <div className="mt-1 text-xs text-muted-foreground">+5.8% vs last 30d</div>
            </div>
            <p className="text-xs text-muted-foreground">
              Section heading bar + content directly on the canvas. The bottom hairline of the
              heading does the visual separation work — no card wrap needed.
            </p>
          </div>

          <div>
            <div className="flex items-baseline justify-between border-b border-border pb-2">
              <h3 className="text-sm font-semibold">Bordered card (rare)</h3>
              <span className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                Only for interactive items
              </span>
            </div>
            <Surface className="mt-4 p-4">
              <div className="flex items-baseline justify-between">
                <div className="text-sm font-semibold">Scale plan</div>
                <StatusTag tone="success">Active</StatusTag>
              </div>
              <div className="num-display mt-2 text-3xl tabular">
                $2,800{" "}
                <span className="ml-1 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                  / yr
                </span>
              </div>
            </Surface>
            <p className="mt-4 text-xs text-muted-foreground">
              Reserve <code className="font-mono text-foreground">{`<Surface>`}</code> for content
              that acts on its own — a subscription tile, a navigable card. Sibling content in a
              list shouldn&apos;t each get a card.
            </p>
          </div>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Primitive"
          title="KpiRow"
          subtitle="Borderless metric row. Cells separated by space + typography only. CLI/Linear flavor."
        />
        <KpiRow items={KPI_DEMO} />
        <p className="text-xs text-muted-foreground">
          <code className="font-mono text-foreground">{`<KpiRow items={...} cols={4} />`}</code> —
          eyebrow + .num-display value + delta/sub line. No outer border, no internal vertical
          hairlines.
        </p>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Primitive"
          title="SectionHead"
          subtitle="The heading bar above borderless sections. Optional eyebrow / action."
        />
        <div className="grid grid-cols-1 gap-x-10 gap-y-6 md:grid-cols-2">
          <div>
            <SectionHead title="MRR over time" subtitle="Trailing 12 months" />
            <p className="mt-3 text-xs text-muted-foreground">Default.</p>
          </div>
          <div>
            <SectionHead
              eyebrow="Insights"
              title="Cohort retention"
              subtitle="% active N months after join"
              action={
                <Button size="xs" variant="outline">
                  Export
                </Button>
              }
            />
            <p className="mt-3 text-xs text-muted-foreground">With eyebrow + action slot.</p>
          </div>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Primitive"
          title="StatusTag"
          subtitle="Compact mono-uppercase tag with optional dot. The canonical status pill."
        />
        <div className="flex flex-wrap items-center gap-2">
          <StatusTag tone="success">Active</StatusTag>
          <StatusTag tone="info">Pending</StatusTag>
          <StatusTag tone="warn">Past due</StatusTag>
          <StatusTag tone="danger">Failed</StatusTag>
          <StatusTag tone="neutral">Refunded</StatusTag>
          <StatusTag tone="success" withDot={false}>
            No dot
          </StatusTag>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Primitive"
          title="DataTable"
          subtitle="Table sits directly on the page background. Hairline horizontal rows only — no card wrap."
        />
        <DataTable columns={TX_COLS} rows={TABLE_SAMPLE} />
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead eyebrow="Components" title="Buttons" subtitle="Six variants × five sizes." />
        <div className="flex flex-wrap items-center gap-2">
          <Button>Primary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="destructive">Destructive</Button>
          <Separator orientation="vertical" className="mx-1 h-6" />
          <Button size="xs">
            <CalendarDays data-icon="inline-start" />
            Range
          </Button>
          <Button size="sm">
            <Download data-icon="inline-start" />
            Export
          </Button>
          <Button size="default">
            <CreditCard data-icon="inline-start" />
            Charge
          </Button>
          <Button size="lg">
            <Check data-icon="inline-start" />
            Approve
          </Button>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead eyebrow="Components" title="Form controls & badges" />
        <div className="grid grid-cols-1 gap-x-10 gap-y-5 md:grid-cols-2">
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs" htmlFor="ds-name">
              Customer name
            </Label>
            <Input id="ds-name" name="ds-name" placeholder="Acme Industries" />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs" htmlFor="ds-search">
              Search
            </Label>
            <div className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                id="ds-search"
                name="ds-search"
                placeholder="Search customers, invoices…"
                className="pl-8"
              />
            </div>
          </div>
          <div className="flex items-center justify-between gap-4">
            <div>
              <div className="text-sm font-medium">Send dunning emails</div>
              <div className="text-xs text-muted-foreground">
                Notify customers on failed charges
              </div>
            </div>
            <Switch defaultChecked />
          </div>
          <div className="flex items-center justify-between gap-4">
            <div>
              <div className="text-sm font-medium">Test mode</div>
              <div className="text-xs text-muted-foreground">Route to the sandbox processor</div>
            </div>
            <Switch />
          </div>
        </div>

        <div className="mt-2 flex flex-wrap items-center gap-2">
          <Badge variant="outline">Neutral</Badge>
          <Badge className="bg-brand-soft text-brand">Brand</Badge>
          <Badge variant="info">Info</Badge>
          <Badge variant="success">Success</Badge>
          <Badge variant="warning">Warning</Badge>
          <Badge variant="destructive">Destructive</Badge>
          <Badge variant="muted" className="font-mono">
            cus_8a3kf2
          </Badge>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Patterns"
          title="Charts"
          subtitle="Monochromatic palette — brand + warm complement + slate scale. Never rainbow."
        />
        <div className="grid grid-cols-1 gap-x-10 gap-y-6 lg:grid-cols-[1fr_1.4fr]">
          <ul role="list" className="flex flex-col">
            {[
              { v: "--chart-1", use: "Primary series · brand" },
              { v: "--chart-2", use: "Secondary series · warm complement" },
              { v: "--chart-3", use: "Tertiary · light brand" },
              { v: "--chart-4", use: "Quaternary · slate" },
              { v: "--chart-5", use: "Background fill · light slate" },
            ].map((c, i) => (
              <li
                key={c.v}
                className={
                  i !== 4
                    ? "flex items-center gap-3 border-b border-border py-2.5"
                    : "flex items-center gap-3 py-2.5"
                }
              >
                <span
                  className="h-5 w-12 shrink-0 rounded-sm border border-border"
                  style={{ background: `var(${c.v})` }}
                />
                <span className="flex-1 text-sm text-foreground">{c.use}</span>
                <code className="font-mono text-[11px] text-muted-foreground">{c.v}</code>
              </li>
            ))}
          </ul>

          <div className="h-56">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={MRR_SERIES} margin={{ top: 10, right: 12, bottom: 0, left: -10 }}>
                <defs>
                  <linearGradient id="ds-grad-1" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="var(--chart-1)" stopOpacity={0.3} />
                    <stop offset="100%" stopColor="var(--chart-1)" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="ds-grad-2" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="var(--chart-2)" stopOpacity={0.22} />
                    <stop offset="100%" stopColor="var(--chart-2)" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid vertical={false} stroke="var(--border)" strokeDasharray="3 3" />
                <XAxis
                  dataKey="month"
                  stroke="var(--muted-foreground)"
                  tick={{ fontSize: 11 }}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  stroke="var(--muted-foreground)"
                  tick={{ fontSize: 11 }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(v) => `${(v / 1000).toFixed(0)}k`}
                  width={36}
                />
                <Area
                  type="monotone"
                  dataKey="mrr"
                  stroke="var(--chart-1)"
                  strokeWidth={1.75}
                  fill="url(#ds-grad-1)"
                  isAnimationActive={false}
                />
                <Area
                  type="monotone"
                  dataKey="expansion"
                  stroke="var(--chart-2)"
                  strokeWidth={1.25}
                  fill="url(#ds-grad-2)"
                  isAnimationActive={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead eyebrow="Tokens" title="Spacing & radii" />
        <div className="grid grid-cols-1 gap-x-10 gap-y-6 md:grid-cols-2">
          <div>
            <div className="mb-3 text-sm font-semibold">Spacing scale</div>
            <ul role="list" className="flex flex-col gap-1.5">
              {SPACING.map((s) => (
                <li key={s.name} className="flex items-center gap-3 text-xs">
                  <span className="w-14 font-mono text-muted-foreground">
                    {s.name} · {s.px}px
                  </span>
                  <span className="h-1.5 bg-brand" style={{ width: s.px }} />
                </li>
              ))}
            </ul>
          </div>
          <div>
            <div className="mb-3 text-sm font-semibold">Radii</div>
            <div className="grid grid-cols-4 gap-4">
              {[{ name: "sm" }, { name: "md" }, { name: "lg" }, { name: "xl" }].map((r) => (
                <div key={r.name} className="flex flex-col items-center gap-1.5">
                  <div className={`size-12 bg-muted rounded-${r.name}`} />
                  <div className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                    {r.name}
                  </div>
                </div>
              ))}
            </div>
            <p className="mt-3 text-xs text-muted-foreground">
              Precision · 6px base radius. Modern but not pillowy.
            </p>
          </div>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead
          eyebrow="Pattern"
          title="Topbar"
          subtitle="Search input + status chip + actions."
        />
        <div className="flex items-center gap-3 border border-border bg-background px-3 py-2">
          <Search className="size-4 text-muted-foreground" />
          <input
            aria-label="Search anything"
            placeholder="Search anything…"
            className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
          <kbd className="rounded border border-border bg-background px-1 py-0.5 font-mono text-[10px] text-muted-foreground">
            ⌘K
          </kbd>
          <Separator orientation="vertical" className="h-5" />
          <span className="inline-flex h-7 items-center gap-1.5 rounded-md border border-warning/30 bg-warning/10 px-2 font-mono text-[10px] uppercase tracking-wider text-warning">
            <span className="size-1.5 rounded-full bg-warning" />
            Test mode
          </span>
          <Button variant="ghost" size="icon-sm" aria-label="Notifications">
            <Bell className="size-4" />
          </Button>
          <Button size="sm">Create</Button>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHead eyebrow="See it live" title="The dashboard uses every token on this page" />
        <div className="flex flex-wrap items-center gap-3">
          <Button asChild>
            <Link href="/dashboard">Open the dashboard</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link href="/customers">See customers</Link>
          </Button>
        </div>
      </section>
    </div>
  );
}

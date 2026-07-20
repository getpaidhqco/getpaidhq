"use client";

import * as React from "react";
import { ChevronRight } from "lucide-react";
import { format } from "date-fns";
import { useQuery } from "@tanstack/react-query";

import { useAuth } from "@getpaidhq/auth";
import { fetchData } from "@/app/(dashboard)/payments/data/data";
import { SectionHead } from "@/components/ui/section-head";
import { StatusTag, type StatusTone } from "@/components/ui/status-tag";
import { formatCurrency } from "@/lib/currency";
import type { PaymentResponse as Payment } from "@getpaidhq/sdk";

const STATUS: Record<string, { label: string; tone: StatusTone }> = {
  succeeded: { label: "Succeeded", tone: "success" },
  pending: { label: "Pending", tone: "info" },
  processing: { label: "Pending", tone: "info" },
  failed: { label: "Failed", tone: "danger" },
  refunded: { label: "Refunded", tone: "neutral" },
};

export function TransactionsTable() {
  const { getAuthHeaders } = useAuth();

  const query = useQuery({
    queryKey: ["recent-payments"],
    queryFn: async () => {
      const headers = await getAuthHeaders();
      const result = await fetchData({ pageIndex: 0, pageSize: 8 }, headers);
      return result;
    },
  });

  const rows: Payment[] = query.data?.data ?? [];

  return (
    <section className="flex flex-col gap-3">
      <SectionHead
        title="Recent transactions"
        subtitle="Latest payments across all currencies"
        action={
          <a
            href="/payments"
            className="inline-flex items-center gap-1 font-mono text-[10px] uppercase tracking-wider text-muted-foreground hover:text-foreground"
          >
            View all <ChevronRight className="size-3" />
          </a>
        }
      />

      <div className="overflow-x-auto whitespace-nowrap">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
              <th className="py-2 pr-3 font-medium">Date</th>
              <th className="px-3 py-2 font-medium">Reference</th>
              <th className="px-3 py-2 text-right font-medium">Amount</th>
              <th className="px-3 py-2 font-medium">Status</th>
              <th className="py-2 pl-3 font-medium">PSP</th>
            </tr>
          </thead>
          <tbody>
            {query.isLoading ? (
              <tr>
                <td
                  colSpan={5}
                  className="px-3 py-8 text-center text-xs text-muted-foreground"
                >
                  Loading…
                </td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td
                  colSpan={5}
                  className="px-3 py-8 text-center text-xs text-muted-foreground"
                >
                  No transactions yet.
                </td>
              </tr>
            ) : (
              rows.map((t, i) => {
                const status = STATUS[t.status] ?? {
                  label: t.status,
                  tone: "neutral" as StatusTone,
                };
                let dateLabel = t.created_at;
                try {
                  dateLabel = format(new Date(t.created_at), "MMM d · HH:mm");
                } catch {
                  /* keep raw */
                }
                return (
                  <tr
                    key={t.id}
                    className={
                      i !== rows.length - 1
                        ? "border-b border-border transition hover:bg-muted/40"
                        : "transition hover:bg-muted/40"
                    }
                  >
                    <td className="py-2.5 pr-3 font-mono text-xs text-muted-foreground">
                      {dateLabel}
                    </td>
                    <td className="px-3 py-2.5">
                      <div className="font-mono text-xs text-foreground">{t.reference}</div>
                    </td>
                    <td className="px-3 py-2.5 text-right">
                      <span className="font-medium tabular text-foreground">
                        {formatCurrency(t.currency, t.amount)}
                      </span>
                      <span className="ml-1 font-mono text-[10px] uppercase text-muted-foreground">
                        {t.currency}
                      </span>
                    </td>
                    <td className="px-3 py-2.5">
                      <StatusTag tone={status.tone}>{status.label}</StatusTag>
                    </td>
                    <td className="py-2.5 pl-3 font-mono text-xs text-muted-foreground">
                      {t.psp_id}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}

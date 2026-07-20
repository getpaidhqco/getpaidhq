"use client";

import { type Column, DataTable } from "@/components/ui/data-table";
import { useInvoice } from "../invoice-context";
import type { InvoiceLineItem } from "@getpaidhq/sdk";
import { formatCurrency } from "@/lib/currency";

export default function ItemsTable() {
  const { invoice } = useInvoice();

  if (!invoice || !invoice.line_items || invoice.line_items.length === 0) {
    return (
      <div className="py-4 text-center text-sm text-muted-foreground">
        No items on this invoice.
      </div>
    );
  }

  const currency = invoice.currency;

  const columns: Column<InvoiceLineItem>[] = [
    {
      key: "description",
      header: "Description",
      render: (item) => <span className="font-medium text-foreground">{item.description}</span>,
    },
    {
      key: "quantity",
      header: "Qty",
      align: "right",
      width: "80px",
      render: (item) => <span className="tabular">{item.quantity}</span>,
    },
    {
      key: "unit_amount",
      header: "Unit price",
      align: "right",
      width: "120px",
      render: (item) => (
        <span className="tabular text-muted-foreground">
          {formatCurrency(currency, Number(item.unit_amount))}
        </span>
      ),
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      width: "120px",
      render: (item) => <span className="tabular">{formatCurrency(currency, item.total)}</span>,
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <DataTable columns={columns} rows={invoice.line_items} />
      <div className="flex justify-end">
        <dl className="flex w-full max-w-xs flex-col gap-2 text-sm">
          <div className="flex justify-between">
            <dt className="text-muted-foreground">Subtotal</dt>
            <dd className="tabular">{formatCurrency(currency, invoice.subtotal)}</dd>
          </div>
          <div className="flex justify-between border-t border-border pt-2 font-semibold">
            <dt>Total</dt>
            <dd className="tabular">{formatCurrency(currency, invoice.total)}</dd>
          </div>
        </dl>
      </div>
    </div>
  );
}

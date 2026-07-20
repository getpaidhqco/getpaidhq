"use client";

import * as React from "react";

import { cn } from "@/lib/utils";
import {
  Pagination,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";

export type Column<Row> = {
  key: string;
  header: string;
  align?: "left" | "right";
  width?: string;
  render: (row: Row) => React.ReactNode;
};

/**
 * Bordered list table — used by every revenue/customer list screen.
 *
 * Sits directly on the page background, no card around it.
 * Hairline horizontal dividers only.
 */
export function DataTable<Row extends { id: string | number }>({
  columns,
  rows,
  empty,
  onRowClick,
  className,
}: {
  columns: Column<Row>[];
  rows: Row[];
  empty?: React.ReactNode;
  onRowClick?: (row: Row) => void;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "-mx-2 overflow-x-auto whitespace-nowrap sm:-mx-4 lg:-mx-6",
        className,
      )}
    >
      <div className="inline-block min-w-full px-2 align-middle sm:px-4 lg:px-6">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/40 text-left font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
              {columns.map((c) => (
                <th
                  key={c.key}
                  className={cn(
                    "py-2 px-3 font-medium whitespace-nowrap",
                    c.align === "right" && "text-right",
                  )}
                  style={c.width ? { width: c.width } : undefined}
                >
                  {c.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td
                  colSpan={columns.length}
                  className="px-3 py-10 text-center text-sm text-muted-foreground"
                >
                  {empty ?? "Nothing to show."}
                </td>
              </tr>
            ) : (
              rows.map((row, i) => (
                <tr
                  key={row.id}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                  className={cn(
                    "transition",
                    onRowClick && "cursor-pointer",
                    i !== rows.length - 1 && "border-b border-border",
                    "hover:bg-muted/40",
                  )}
                >
                  {columns.map((c) => (
                    <td
                      key={c.key}
                      className={cn(
                        "px-3 py-2.5 align-middle",
                        c.align === "right" && "text-right",
                      )}
                    >
                      {c.render(row)}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

/**
 * {@link DataTable} with simple client-side pagination. The full `rows` array is
 * sliced to `pageSize` (default 10) and prev/next controls appear only when the
 * data spans more than one page. Use for in-page detail tables that already have
 * all their rows in memory.
 */
export function PaginatedDataTable<Row extends { id: string | number }>({
  columns,
  rows,
  pageSize = 10,
  empty,
  onRowClick,
  className,
}: {
  columns: Column<Row>[];
  rows: Row[];
  pageSize?: number;
  empty?: React.ReactNode;
  onRowClick?: (row: Row) => void;
  className?: string;
}) {
  const [page, setPage] = React.useState(0);

  const pageCount = Math.max(1, Math.ceil(rows.length / pageSize));
  // Clamp in case `rows` shrinks below the current page (e.g. after a refetch).
  const current = Math.min(page, pageCount - 1);
  const start = current * pageSize;
  const pageRows = rows.slice(start, start + pageSize);

  return (
    <div className="flex flex-col gap-4">
      <DataTable
        columns={columns}
        rows={pageRows}
        empty={empty}
        onRowClick={onRowClick}
        className={className}
      />
      {pageCount > 1 ? (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Page {current + 1} of {pageCount}
          </p>
          <Pagination variant="compact">
            <PaginationPrevious
              size="sm"
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={current === 0}
            />
            <PaginationNext
              size="sm"
              onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
              disabled={current >= pageCount - 1}
            />
          </Pagination>
        </div>
      ) : null}
    </div>
  );
}

/**
 * Toolbar above a DataTable. Search input on the left, filters/actions on the right.
 */
export function TableToolbar({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-2 border-b border-border pb-3",
        className,
      )}
    >
      {children}
    </div>
  );
}

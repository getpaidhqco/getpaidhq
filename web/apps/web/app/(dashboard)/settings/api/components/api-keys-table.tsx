"use client"

import { useState } from "react"
import { Trash } from "lucide-react"
import { useQuery } from "@tanstack/react-query"
import { format } from "date-fns"
import { toast } from "sonner"

import { useGetPaidHQClient } from "@getpaidhq/react-sdk"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { type Column, DataTable } from "@/components/ui/data-table"

// The list envelope returns api-key records without the secret `key` field;
// these are the fields the SDK ApiKeyCreateResponse shares with stored keys.
export type ApiKey = {
  id: string
  name?: string | null
  created_at: string
  updated_at: string
}

export function ApiKeysTable() {
  const client = useGetPaidHQClient()
  const [toRevoke, setToRevoke] = useState<ApiKey | null>(null)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ["api-keys"],
    queryFn: () => client.apiKeys.list(),
  })

  const apiKeys = (data?.data as ApiKey[]) ?? []

  const revokeKey = async (id: string) => {
    try {
      await client.apiKeys.delete(id)
      toast.success("API key revoked")
      refetch()
    } catch {
      toast.error("Failed to revoke API key")
    } finally {
      setToRevoke(null)
    }
  }

  const columns: Column<ApiKey>[] = [
    {
      key: "name",
      header: "Name",
      render: (k) => (
        <span className="text-sm text-foreground">
          {k.name || <span className="text-muted-foreground">Unnamed key</span>}
        </span>
      ),
    },
    {
      key: "id",
      header: "Key ID",
      render: (k) => (
        <span className="font-mono text-xs text-muted-foreground">{k.id}</span>
      ),
    },
    {
      key: "created_at",
      header: "Created",
      render: (k) => (
        <span className="font-mono text-xs tabular text-muted-foreground">
          {format(new Date(k.created_at), "MMM d, yyyy")}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      width: "48px",
      render: (k) => (
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={(e) => {
            e.stopPropagation()
            setToRevoke(k)
          }}
          aria-label="Revoke API key"
        >
          <Trash className="size-3.5 text-destructive" />
        </Button>
      ),
    },
  ]

  if (isLoading) {
    return (
      <div className="py-6 text-center text-sm text-muted-foreground">
        Loading API keys…
      </div>
    )
  }

  return (
    <>
      <DataTable
        columns={columns}
        rows={apiKeys}
        empty={
          <div className="py-6 text-center text-sm text-muted-foreground">
            No API keys yet. Create one to authenticate API requests.
          </div>
        }
      />

      <AlertDialog
        open={!!toRevoke}
        onOpenChange={(o) => {
          if (!o) setToRevoke(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke API key?</AlertDialogTitle>
            <AlertDialogDescription>
              {toRevoke ? (
                <>
                  Any request using{" "}
                  <code className="font-mono text-xs">
                    {toRevoke.name || toRevoke.id}
                  </code>{" "}
                  will immediately stop working. This can't be undone.
                </>
              ) : (
                "This action cannot be undone."
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => toRevoke && revokeKey(toRevoke.id)}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              Revoke key
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

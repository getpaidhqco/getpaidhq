"use client"

import { Button } from "@/components/ui/button";
import { Copy } from "lucide-react";
import { toast } from "sonner";

interface ApiKeyCopyButtonProps {
  apiKey: string;
}

export function ApiKeyCopyButton({ apiKey }: ApiKeyCopyButtonProps) {
  const copyApiKey = () => {
    navigator.clipboard.writeText(apiKey);
    toast.success("API key copied", {
      description: "The API key has been copied to your clipboard.",
    });
  };

  return (
    <Button variant="outline" size="icon-sm" onClick={copyApiKey} aria-label="Copy API key">
      <Copy className="size-3.5" />
    </Button>
  );
}

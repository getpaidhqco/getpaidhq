import {Badge} from "@/components/ui/badge";
import clsx from "clsx";


export default function GrowthBadge({growth}: {growth: number}) {
  const growthColor = growth >= 0 ? "bg-emerald-500/24 text-emerald-500" : "bg-rose-500/24 text-rose-500 ";

  return (
    <Badge className={clsx("mt-1.5 border-none", growthColor)}>
      {growth >= 0 ? `+${parseFloat(growth.toFixed(2))}%` : `${parseFloat(growth.toFixed(2))}%`}
    </Badge>
  );
}

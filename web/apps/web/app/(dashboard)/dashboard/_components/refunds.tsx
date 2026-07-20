"use client";

import {useEffect, useId, useState} from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  Rectangle,
  XAxis,
  YAxis,
} from "recharts";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
} from "@/components/ui/chart";
import { CustomTooltipContent } from "./charts-extra";
import { Badge } from "@/components/ui/badge";
import {useAuth} from "@getpaidhq/auth";
import {RecurringRevenue} from "@/lib/schemas/recurring-revenue";
import {keepPreviousData, useQuery} from "@tanstack/react-query";
import {fetchActiveSubscribers, fetchRefunds} from "@/app/(dashboard)/dashboard/data";
import {format} from "date-fns";
import GrowthBadge from "@/app/(dashboard)/dashboard/_components/growth-badge";

const chartConfig = {
  total: {
    label: "Total",
    color: "var(--chart-4)",
  },
  projected: {
    label: "Projected",
    color: "var(--chart-3)",
  },
} satisfies ChartConfig;

interface CustomCursorProps {
  fill?: string;
  pointerEvents?: string;
  height?: number;
  points?: Array<{ x: number; y: number }>;
  className?: string;
}

function CustomCursor(props: CustomCursorProps) {
  const { fill, pointerEvents, height, points, className } = props;

  if (!points || points.length === 0) {
    return null;
  }

  const { x, y } = points[0]!;
  return (
    <>
      <Rectangle
        x={x - 12}
        y={y}
        fill={fill}
        pointerEvents={pointerEvents}
        width={24}
        height={height}
        className={className}
        type="linear"
      />
      <Rectangle
        x={x - 1}
        y={y}
        fill={fill}
        pointerEvents={pointerEvents}
        width={1}
        height={height}
        className="recharts-tooltip-inner-cursor"
        type="linear"
      />
    </>
  );
}
type Props = {
  startDate: Date;
  endDate: Date;
}


export function Refunds({startDate, endDate}: Props) {
  const id = useId();
  const {getAuthHeaders} = useAuth()
  const [chartData, setChartData] = useState<RecurringRevenue[]>([])
  const [total, setTotal] = useState<number>()
  const [growth, setGrowth] = useState<number>(0)

  const query = useQuery<RecurringRevenue[] | undefined>({
    queryKey: ['refunds', startDate, endDate],
    queryFn: async () => fetchRefunds({
      startDate: startDate.toISOString().split("T")[0],
      endDate: endDate.toISOString().split("T")[0],
      authHeaders: await getAuthHeaders()
    }),
    placeholderData: keepPreviousData,
  })

  useEffect(() => {
    if (query.data) {
      setChartData(query?.data);
      setTotal(query.data.reduce((sum, item) => sum + parseFloat(item.total.toFixed(2)), 0))
      setGrowth(query.data[query.data.length - 1]?.growth_mom || 0);
    }
  }, [query.data]);



  return (
    <Card className="gap-4">
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="space-y-0.5">
            <CardTitle>Refunds</CardTitle>
            <div className="flex items-start gap-2">
              <div className="font-semibold text-2xl">${total}</div>
              <GrowthBadge growth={growth}/>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex items-center gap-2">
              <div
                aria-hidden="true"
                className="size-1.5 shrink-0 rounded-xs bg-chart-4"
              ></div>
              <div className="text-[13px]/3 text-muted-foreground/50">
                total
              </div>
            </div>
            <div className="flex items-center gap-2">
              <div
                aria-hidden="true"
                className="size-1.5 shrink-0 rounded-xs bg-chart-3"
              ></div>
              <div className="text-[13px]/3 text-muted-foreground/50">
                Projected
              </div>
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <ChartContainer
          config={chartConfig}
          className="aspect-auto h-60 w-full [&_.recharts-rectangle.recharts-tooltip-cursor]:fill-(--chart-4)/10 [&_.recharts-rectangle.recharts-tooltip-inner-cursor]:fill-white/20"
        >
          <LineChart
            accessibilityLayer
            data={chartData}
            margin={{ left: -12, right: 12, top: 12 }}
          >
            <defs>
              <linearGradient id={`${id}-gradient`} x1="0" y1="0" x2="1" y2="0">
                <stop offset="0%" stopColor="var(--chart-5)" />
                <stop offset="100%" stopColor="var(--chart-4)" />
              </linearGradient>
            </defs>
            <CartesianGrid
              vertical={false}
              strokeDasharray="2 2"
              stroke="var(--border)"
            />
            <XAxis
              dataKey="period"
              tickLine={false}
              tickMargin={12}
              tickFormatter={(value) => {
                return format(value, "MMM yyyy")
              }}
              stroke="var(--border)"
            />
            <YAxis
              axisLine={false}
              tickLine={false}
              tickFormatter={(value) => {
                if (value === 0) return "$0";
                return `$${value / 1000}k`;
              }}
              interval="preserveStartEnd"
            />
            <Line
              type="linear"
              dataKey="projected"
              stroke="var(--color-projected)"
              strokeWidth={2}
              dot={false}
              activeDot={false}
            />
            <ChartTooltip
              content={
                <CustomTooltipContent
                  colorMap={{
                    total: "var(--chart-4)",
                    projected: "var(--chart-3)",
                  }}
                  labelMap={{
                    total: "total",
                    projected: "Projected",
                  }}
                  dataKeys={["total", "projected"]}
                  valueFormatter={(value) => `$${value.toLocaleString()}`}
                />
              }
              cursor={<CustomCursor fill="var(--chart-4)" />}
            />
            <Line
              type="linear"
              dataKey="total"
              stroke={`url(#${id}-gradient)`}
              strokeWidth={2}
              dot={false}
              activeDot={{
                r: 5,
                fill: "var(--chart-4)",
                stroke: "var(--background)",
                strokeWidth: 2,
              }}
            />
          </LineChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

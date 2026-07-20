"use client";

import {useEffect, useId, useState} from "react";
import {Bar, BarChart, CartesianGrid, XAxis, YAxis} from "recharts";

import {Card, CardContent, CardHeader, CardTitle} from "@/components/ui/card";
import {ChartConfig, ChartContainer, ChartTooltip,} from "@/components/ui/chart";
import {CustomTooltipContent} from "./charts-extra";
import {RadioGroup, RadioGroupItem} from "@/components/ui/radio-group";
import {keepPreviousData, useQuery} from "@tanstack/react-query";
import {fetchMrrArrData} from "@/app/(dashboard)/dashboard/data";
import {useAuth} from "@getpaidhq/auth";
import {RecurringRevenue} from "@/lib/schemas/recurring-revenue";
import {formatCurrency} from "@/lib/currency";
import {format} from "date-fns";
import GrowthBadge from "@/app/(dashboard)/dashboard/_components/growth-badge";

const chartConfig = {
  total: {
    label: "total",
    color: "var(--chart-1)",
  },
  projected: {
    label: "Projected",
    color: "var(--chart-3)",
  },
} satisfies ChartConfig;

type Props = {
  startDate: Date;
  endDate: Date;
}

export function ChartMrr({startDate, endDate}: Props) {
  const id = useId();
  const {getAuthHeaders} = useAuth()
  const [type, setType] = useState<string | "arr" | "mrr">("mrr");
  const [chartData, setChartData] = useState<RecurringRevenue[]>([])
  const [total, setTotal] = useState<number>()
  const [growth, setGrowth] = useState<number>(0)

  const mrrQuery = useQuery<RecurringRevenue[] | undefined>({
    queryKey: [type, startDate, endDate],
    queryFn: async () => fetchMrrArrData({
      type,
      startDate: startDate.toISOString().split("T")[0],
      endDate: endDate.toISOString().split("T")[0],
      authHeaders: await getAuthHeaders()
    }),
    placeholderData: keepPreviousData, // don't have 0 rows flash while changing pages/loading next page
  })

  useEffect(() => {
    if (mrrQuery.data) {

      setChartData(mrrQuery?.data);
      setTotal(parseFloat((mrrQuery.data[mrrQuery.data.length - 1]?.total || 0).toFixed(2)));
      setGrowth(mrrQuery.data[mrrQuery.data.length - 1]?.growth_mom || 0);
    }
  }, [mrrQuery.data]);


  // use this as X-axis ticks to show only first and last tick
  // const firstMonth = format(startDate, 'PP');
  // const last = chartData[chartData.length - 1]?.period as string;
  // const lastMonth = last ? format(new Date(last), 'PP') : format(endDate, 'PP');



  return (
    <Card className="gap-4">
      <CardHeader>
        <div className="flex items-center justify-between gap-2">
          <div className="space-y-0.5">
            <CardTitle>Recurring Revenue</CardTitle>
            <div className="flex items-start gap-2">
              <div className="font-semibold text-2xl">
                {formatCurrency("USD", total)}
              </div>
              <GrowthBadge growth={growth}/>
            </div>
          </div>
          <div className="bg-black/10 dark:bg-black/50  inline-flex h-7 rounded-lg p-0.5 shrink-0">
            <RadioGroup
              value={type}
              onValueChange={setType}
              className="group text-xs after:border after:border-border after:bg-background has-focus-visible:after:border-ring has-focus-visible:after:ring-ring/50 relative inline-grid grid-cols-[1fr_1fr] items-center gap-0 font-medium after:absolute after:inset-y-0 after:w-1/2 after:rounded-md after:shadow-xs after:transition-[translate,box-shadow] after:duration-300 after:[transition-timing-function:cubic-bezier(0.16,1,0.3,1)] has-focus-visible:after:ring-[3px] data-[state=mrr]:after:translate-x-0 data-[state=arr]:after:translate-x-full"
              data-state={type}
            >
              <label
                className="group-data-[state=arr]:text-muted-foreground/50 relative z-10 inline-flex h-full min-w-8 cursor-pointer items-center justify-center px-2 whitespace-nowrap transition-colors select-none">
                MRR
                <RadioGroupItem
                  id={`${id}-1`}
                  value="mrr"
                  className="sr-only"
                />
              </label>
              <label
                className="group-data-[state=mrr]:text-muted-foreground/50 relative z-10 inline-flex h-full min-w-8 cursor-pointer items-center justify-center px-2 whitespace-nowrap transition-colors select-none">
                ARR
                <RadioGroupItem id={`${id}-2`} value="arr" className="sr-only"/>
              </label>
            </RadioGroup>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <ChartContainer
          config={chartConfig}
          className="aspect-auto h-60 w-full [&_.recharts-rectangle.recharts-tooltip-cursor]:fill-[var(--chart-1)]/15"
        >
          <BarChart
            accessibilityLayer
            data={chartData}
            maxBarSize={20}
            margin={{left: -12, right: 12, top: 12}}
          >
            <defs>
              <linearGradient id={`${id}-gradient`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="var(--chart-1)"/>
                <stop offset="100%" stopColor="var(--chart-2)"/>
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
              // ticks={[firstMonth, lastMonth]}
              tickFormatter={(value) => {
                return format(value, "MMM yyyy")
              }}
              stroke="var(--border)"
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              tickFormatter={(value) =>
                value === 0 ? "$0" : `$${(value / 1000).toFixed(1)}K`  // TODO calculate the scale
              }
            />
            <ChartTooltip
              content={
                <CustomTooltipContent
                  colorMap={{
                    total: "var(--chart-1)",
                  }}
                  labelMap={{
                    total: "Total",
                  }}
                  dataKeys={["total"]}
                  valueFormatter={(value) => `$${(value / 100).toLocaleString()}`}
                />
              }
            />
            <Bar dataKey="total" fill={`url(#${id}-gradient)`} stackId="a"/>
            <Bar
              dataKey="projected"
              fill="var(--color-projected)"
              stackId="a"
            />
          </BarChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

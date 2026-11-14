import { Loader2, ShieldAlert } from "lucide-react";
import { useEffect, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";

type Entry = {
  rank: number;
  symbol: string;
  score: number;
  return_pct: string;
  daily_volatility_pct: string;
  last_close: string;
  last_day: string;
};
type Metrics = {
  days: number;
  total_return_pct: string;
  benchmark_return_pct: string;
  max_drawdown_pct: string;
  sharpe: number;
  hit_rate_pct: string;
  trades: number;
  stops_hit: number;
  fees_paid: string;
  slippage_cost: string;
  costs_pct_of_initial: string;
};
type MarketReport = {
  market: string;
  source: string;
  currency: string;
  universe: number;
  ranking: { top: Entry[]; eligible: number; skipped: { symbol: string; reason: string }[] };
  backtest?: { from: string; to: string; metrics: Metrics };
  error?: string;
};
export type RecommendationReport = {
  date: string;
  generated_at: string;
  disclaimer: string;
  criterion: string;
  strategy: string;
  b3: string;
  markets: MarketReport[];
};

const marketNames: Record<string, string> = { US: "US stocks", CRYPTO: "Crypto" };

export function Disclaimer({ text }: { text: string }) {
  return (
    <div
      role="note"
      className="flex items-start gap-3 rounded-lg border border-amber-500/40 bg-amber-500/10 p-4 text-sm"
    >
      <ShieldAlert className="mt-0.5 size-4 shrink-0 text-amber-500" aria-hidden="true" />
      <p>{text}</p>
    </div>
  );
}

function BacktestSummary({ m, from, to }: { m: Metrics; from: string; to: string }) {
  const cells: [string, string][] = [
    ["Return", `${m.total_return_pct}%`],
    ["Equal-weight buy and hold", `${m.benchmark_return_pct}%`],
    ["Max drawdown", `${m.max_drawdown_pct}%`],
    ["Sharpe (annualized)", String(m.sharpe)],
    ["Hit rate", `${m.hit_rate_pct}%`],
    ["Closed trades", `${m.trades} (${m.stops_hit} stops)`],
    ["Costs", `${m.costs_pct_of_initial}% of capital`],
  ];
  return (
    <div>
      <p className="mb-2 text-xs text-muted-foreground">
        Replay of past bars from {from} to {to}, after fees and slippage. Not broker execution and
        no promise of profit.
      </p>
      <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {cells.map(([label, value]) => (
          <div key={label}>
            <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</dt>
            <dd className="font-mono text-sm">{value}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

function MarketCard({ m }: { m: MarketReport }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {marketNames[m.market] ?? m.market}: top {m.ranking.top.length}
        </CardTitle>
        <CardDescription>
          {m.source}. {m.ranking.eligible} of {m.universe} symbols eligible.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {m.error ? (
          <p role="alert" className="text-sm text-destructive">
            Unavailable in the last run: {m.error}
          </p>
        ) : (
          <div className="overflow-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  {["#", "Symbol", "Return", "Daily volatility", "Score", "Last close"].map((h) => (
                    <TableHead key={h} className="text-[11px] uppercase tracking-wide">
                      {h}
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {m.ranking.top.map((e) => (
                  <TableRow key={e.symbol}>
                    <TableCell>{e.rank}</TableCell>
                    <TableCell className="font-mono">{e.symbol}</TableCell>
                    <TableCell className="font-mono">{e.return_pct}%</TableCell>
                    <TableCell className="font-mono">{e.daily_volatility_pct}%</TableCell>
                    <TableCell className="font-mono">{e.score.toFixed(2)}</TableCell>
                    <TableCell className="font-mono">
                      {e.last_close} {m.currency}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
        {m.backtest && (
          <BacktestSummary m={m.backtest.metrics} from={m.backtest.from} to={m.backtest.to} />
        )}
      </CardContent>
    </Card>
  );
}

/** Daily top-20 US stocks and top-10 crypto ranking with its backtest; informational only. */
export default function Recommendations() {
  const [report, setReport] = useState<RecommendationReport>();
  const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    api<RecommendationReport>("/api/recommendations/latest", undefined, controller.signal)
      .then(setReport)
      .catch((e) => {
        if (!controller.signal.aborted) setError(e instanceof Error ? e.message : String(e));
      });
    return () => controller.abort();
  }, []);
  if (error)
    return (
      <p role="alert" className="text-sm text-destructive">
        {error}
      </p>
    );
  if (!report)
    return (
      <div className="flex justify-center py-10" role="status" aria-label="Loading">
        <Loader2 className="animate-spin" />
      </div>
    );
  return (
    <div className="space-y-5">
      <Disclaimer text={report.disclaimer} />
      {report.markets.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No ranking yet. The daily job runs at 00:30 UTC; run <code>reco:daily</code> to create the
          first one.
        </p>
      ) : (
        <>
          <p className="text-xs text-muted-foreground">
            Report of {report.date} (generated {new Date(report.generated_at).toLocaleString()}).
          </p>
          <div className="grid gap-5 xl:grid-cols-2">
            {report.markets.map((m) => (
              <MarketCard key={m.market} m={m} />
            ))}
          </div>
        </>
      )}
      <Card>
        <CardHeader>
          <CardTitle>How the ranking works</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <p>{report.criterion}</p>
          <p>{report.strategy}</p>
          <p>{report.b3}</p>
        </CardContent>
      </Card>
      <Disclaimer text={report.disclaimer} />
    </div>
  );
}

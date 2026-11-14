import { Head } from "@inertiajs/react";
import {
  Activity,
  ArrowDownToLine,
  ArrowUpRight,
  CircleHelp,
  Database,
  FlaskConical,
  Loader2,
  Plus,
  RefreshCw,
  ShieldAlert,
  Upload,
  Wifi,
  WifiOff,
} from "lucide-react";
import {
  cloneElement,
  type FormEvent,
  isValidElement,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useState,
} from "react";
import MarketChart from "@/components/MarketChart";
import Recommendations from "@/components/Recommendations";
import { TapeLadder } from "@/components/TapeLadder";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { OrderTicket, SafetyBar, WatchPanel } from "@/components/workbench";
import { api } from "@/lib/api";
import {
  type Account,
  alive,
  type Candle,
  type Dataset,
  date,
  type Job,
  type KindSpec,
  money,
  type ParamSpec,
  type Policy,
  type Preferences,
  type Snapshot,
  type Strategy,
} from "@/types";

const titles: Record<string, [string, string]> = {
  overview: ["Overview", "Your market. Your rules. All decisions in one place."],
  market: ["Market data", "History with identified source. No invented data."],
  strategies: ["Strategies", "Turn hypotheses into deterministic, versioned rules."],
  backtests: ["Lab", "Test, compare and evaluate after costs."],
  orders: ["Orders", "From signal to confirmation. Every transition matters."],
  positions: ["Positions", "Exposure segregated by account, currency and mode."],
  risk: ["Risk control", "Set limits before allowing new exposures."],
  audit: ["Audit", "A persistent trail of decisions and actions."],
  settings: ["Settings", "Status of integrations and the execution environment."],
  recommendations: [
    "Daily ranking",
    "Top US stocks and crypto from end-of-day data. Informational only.",
  ],
};
type NumericKey = "fast" | "slow" | "threshold" | "stop_bps" | "target_bps";
const accountOf = (market: string) =>
  market === "B3" ? "b3" : market === "US" ? "us" : "research";
const firstSymbol: Record<string, string> = { b3: "PETR4", us: "AAPL", research: "BTCUSDT" };
const fallbackParams: ParamSpec[] = [
  { name: "fast", label: "Fast window", type: "int", min: 2, max: 499, step: 1, default: 9 },
  {
    name: "slow",
    label: "Slow window / lookback",
    type: "int",
    min: 3,
    max: 500,
    step: 1,
    default: 21,
  },
  { name: "threshold", label: "Z-score", type: "float", min: 0.1, max: 10, step: 0.1, default: 2 },
  {
    name: "stop_bps",
    label: "Stop (bps)",
    type: "int",
    min: 10,
    max: 2000,
    step: 10,
    default: 100,
  },
  {
    name: "target_bps",
    label: "Target (bps)",
    type: "int",
    min: 10,
    max: 5000,
    step: 10,
    default: 200,
  },
];
const kindName: Record<string, string> = {
  ma_cross: "Moving average crossover",
  mean_reversion: "Mean reversion",
  breakout: "Breakout",
};
function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Label className="grid gap-2 text-xs font-normal text-muted-foreground">
      {label}
      {isValidElement(children)
        ? cloneElement(children as ReactElement<{ "aria-label"?: string }>, { "aria-label": label })
        : children}
    </Label>
  );
}
function Select({
  value,
  onChange,
  children,
  ...props
}: {
  value: string;
  onChange: (v: string) => void;
  children: ReactNode;
  "aria-label"?: string;
}) {
  return (
    <select
      {...props}
      className="h-9 w-full rounded-lg border bg-background px-3 text-sm text-foreground outline-none focus:ring-2 focus:ring-emerald-500"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    >
      {children}
    </select>
  );
}
function Empty({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="flex min-h-48 flex-col items-center justify-center gap-3 p-8 text-center">
      <div className="rounded-xl border bg-muted/30 p-3">
        <Database className="size-5 text-muted-foreground" />
      </div>
      <p className="text-sm font-medium">{title}</p>
      <p className="max-w-md text-xs leading-relaxed text-muted-foreground">{detail}</p>
    </div>
  );
}
function Status({ value }: { value: string }) {
  const good = ["enabled", "completed", "filled", "ready"].includes(value);
  return (
    <Badge
      variant="outline"
      className={good ? "border-emerald-500/20 text-emerald-500" : "text-muted-foreground"}
    >
      {value}
    </Badge>
  );
}
interface GridRow {
  id: string | number;
  cells: ReactNode[];
}

const row = (id: string | number, ...cells: ReactNode[]): GridRow => ({ id, cells });

function GridTable({ headers, rows }: { headers: string[]; rows: GridRow[] }) {
  return rows.length ? (
    <div className="overflow-auto">
      <Table>
        <TableHeader>
          <TableRow>
            {headers.map((h) => (
              <TableHead key={h} className="text-[11px] uppercase tracking-wide">
                {h}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              {r.cells.map((cell, j) => (
                <TableCell key={headers[j]} className="text-xs">
                  {cell}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  ) : (
    <Empty
      title="No records yet"
      detail="Data will appear here after you configure a strategy and run a simulation."
    />
  );
}
function Stat({
  label,
  value,
  detail,
  positive,
}: {
  label: string;
  value: string;
  detail: string;
  positive?: boolean;
}) {
  return (
    <Card>
      <CardContent className="pt-5">
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          {label}
          <ArrowUpRight className="size-3" />
        </div>
        <p
          className={`mt-3 font-mono text-2xl tracking-tight ${positive ? "text-emerald-500" : ""}`}
        >
          {value}
        </p>
        <p className="mt-2 text-[10px] text-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  );
}
export default function Terminal({ section }: { section: string }) {
  const [snapshot, setSnapshot] = useState<Snapshot>();
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [accountId, setAccountState] = useState("b3");
  const [prefs, setPrefs] = useState<Preferences>();
  const [datasetId, setDataset] = useState("");
  const [candles, setCandles] = useState<Candle[]>([]);
  const [report, setReport] = useState<Job>();
  const [reportOpen, setReportOpen] = useState(false);
  const refresh = useCallback(async () => {
    try {
      setSnapshot(await api<Snapshot>("/api/snapshot"));
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Connection failed");
    }
  }, []);
  useEffect(() => {
    void refresh();
    const timer = setInterval(() => {
      if (!document.hidden) void refresh();
    }, 10000);
    return () => clearInterval(timer);
  }, [refresh]);
  useEffect(() => {
    if (typeof EventSource === "undefined") return;
    const source = new EventSource("/api/stream");
    source.addEventListener("changed", () => void refresh());
    return () => source.close();
  }, [refresh]);
  useEffect(() => {
    let active = true;
    api<Preferences>("/api/preferences")
      .then((p) => {
        if (!active) return;
        setPrefs(p);
        setAccountState(p.account);
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);
  const savePrefs = (next: Preferences) => {
    setPrefs(next);
    void api("/api/preferences", next, undefined, "PUT").catch((e) =>
      setError(e instanceof Error ? e.message : "Could not save the workspace"),
    );
  };
  const setAccount = (id: string) => {
    setAccountState(id);
    if (prefs) savePrefs({ ...prefs, account: id });
  };
  const act = async (path: string, body: unknown) => {
    setBusy(true);
    setNotice("");
    try {
      await api(path, body);
      await refresh();
      setNotice("Request recorded.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to run action");
    } finally {
      setBusy(false);
    }
  };
  const placeTicket = (key: string) => {
    if (prefs && (accountId === "b3" || accountId === "us"))
      void act(`/api/accounts/${accountId}/ticket`, { key, symbol: prefs.symbol });
  };
  useEffect(() => {
    if (!prefs) return;
    function onKey(e: KeyboardEvent) {
      const el = e.target as HTMLElement | null;
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      if (el && (el.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName)))
        return;
      if (document.querySelector("[role=dialog]")) return;
      const combo = (e.shiftKey ? "shift+" : "") + e.key.toLowerCase();
      if ((prefs?.templates ?? []).some((t) => t.key === combo)) {
        e.preventDefault();
        placeTicket(combo);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });
  const datasetStamp = snapshot?.datasets.find((d) => d.id === datasetId)?.last ?? "";
  useEffect(() => {
    if (!datasetId) {
      setCandles([]);
      return;
    }
    const controller = new AbortController();
    api<Dataset>(
      `/api/datasets/${datasetId}?v=${encodeURIComponent(datasetStamp)}`,
      undefined,
      controller.signal,
    )
      .then((ds) => setCandles(ds.candles ?? []))
      .catch((e) => {
        if (!controller.signal.aborted) setError(String(e));
      });
    return () => controller.abort();
  }, [datasetId, datasetStamp]);
  const account = snapshot?.accounts.find((a) => a.id === accountId);
  const [title, description] = titles[section] ?? titles.overview;
  if (!snapshot)
    return (
      <>
        <Head title={title} />
        <div className="flex min-h-[50vh] flex-col items-center justify-center gap-4">
          {error ? (
            <>
              <WifiOff />
              <p role="alert">{error}</p>
              <Button onClick={() => void refresh()}>Try again</Button>
            </>
          ) : (
            <>
              <Loader2 className="animate-spin text-emerald-400" />
              <p className="text-sm text-muted-foreground">Loading your workspace…</p>
            </>
          )}
        </div>
      </>
    );
  const currency = account?.currency ?? "BRL";
  const known = account?.id === "b3" || (!!account && !account.reconciled_at.startsWith("0001"));
  const controls = (a: Account) => (
    <div className="flex flex-wrap gap-2">
      <Confirm
        title={a.halted ? "Resume simulation" : "Block new entries"}
        description={
          a.halted
            ? "The engine will keep running without the browser. Review data and limits before resuming."
            : "Blocks new entries; in-flight orders and positions still need to be monitored."
        }
        label={a.halted ? "Resume PAPER" : "Block"}
        disabled={busy}
        action={() =>
          act(`/api/accounts/${a.id}/control`, {
            action: a.halted ? "resume" : "halt",
            confirm: "PAPER",
          })
        }
      />
      {a.id === "b3" && (
        <Confirm
          title="New B3 replay"
          description="Archives the previous snapshot, preserving fills and audit, and resets the simulated capital to R$ 100,000. Requires a blocked account with no open positions or orders."
          label="New replay"
          disabled={busy}
          action={() => act(`/api/accounts/${a.id}/control`, { action: "reset", confirm: "RESET" })}
        />
      )}
      <Confirm
        title="Cancel open orders"
        description="The account will be blocked. External cancellations remain pending until confirmed."
        label="Cancel orders"
        disabled={busy}
        action={() => act(`/api/accounts/${a.id}/control`, { action: "cancel" })}
      />
      {a.id === "us" && (
        <Confirm
          title="Flatten paper positions"
          description="Creates exit orders based on the reconciled position. Does not guarantee immediate execution or closing."
          label="Flatten positions"
          disabled={busy}
          action={() =>
            act(`/api/accounts/${a.id}/control`, { action: "flatten", confirm: "ZERAR" })
          }
        />
      )}
    </div>
  );
  return (
    <>
      <Head title={`${title} · vukem-trading`} />
      <div className="mb-7 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="mb-2 text-[10px] font-medium uppercase tracking-[.2em] text-emerald-500">
            Workspace / {section === "backtests" ? "Quantitative research" : "Trading terminal"}
          </p>
          <h1 className="text-3xl font-semibold tracking-tight">{title}</h1>
          <p className="mt-2 text-sm text-muted-foreground">{description}</p>
        </div>
        <div className="flex items-center gap-2">
          <Select value={accountId} onChange={setAccount} aria-label="Account">
            <option value="b3">Brazil · BRL · Replay</option>
            <option value="us">US · USD · Alpaca Paper</option>
            <option value="research">Research · crypto and FX · backtests only</option>
          </Select>
          <Button size="icon" variant="outline" onClick={() => void refresh()} aria-label="Refresh">
            <RefreshCw className="size-4" />
          </Button>
        </div>
      </div>
      {error && (
        <div
          role="alert"
          className="mb-4 flex justify-between rounded-xl border border-red-500/30 bg-red-500/5 p-4 text-sm text-red-500"
        >
          {error}
          <button type="button" onClick={() => setError("")} aria-label="Close alert">
            ×
          </button>
        </div>
      )}
      {notice && (
        <div
          role="status"
          className="mb-4 rounded-xl border border-emerald-500/20 p-3 text-xs text-emerald-500"
        >
          {notice}
        </div>
      )}
      {account && (
        <div className="mb-6 flex flex-wrap items-center gap-3 rounded-xl border bg-muted/15 px-4 py-3 text-xs">
          <span
            className={`size-1.5 rounded-full ${alive(account.heartbeat) ? "bg-emerald-400" : "bg-amber-400"}`}
          />
          <span>{alive(account.heartbeat) ? "Worker connected" : "Waiting for worker"}</span>
          <span className="text-muted-foreground">/</span>
          <Badge variant="secondary">
            {account.mode === "simulation" ? "B3 SIMULATION · REPLAY" : "ALPACA PAPER · IEX"}
          </Badge>
          <span className="ml-auto text-muted-foreground">
            {account.halted ? account.reason : "New entries enabled"}
          </span>
        </div>
      )}
      {account && <SafetyBar account={account} act={act} busy={busy} />}
      {prefs && snapshot && (section === "overview" || section === "orders") && (
        <div className="mb-6">
          <OrderTicket prefs={prefs} accountId={accountId} place={placeTicket} busy={busy} />
        </div>
      )}
      {prefs && snapshot && section === "market" && (
        <div className="mb-6 grid gap-5 xl:grid-cols-2">
          <WatchPanel
            instruments={snapshot.instruments}
            datasets={snapshot.datasets}
            coverage={snapshot.coverage}
            prefs={prefs}
            save={savePrefs}
          />
          <Card>
            <CardHeader>
              <CardTitle>Tape and ladder</CardTitle>
              <CardDescription>
                Latest closes of the selected dataset and the volume traded at each price level
              </CardDescription>
            </CardHeader>
            <CardContent>
              <TapeLadder candles={candles} />
            </CardContent>
          </Card>
        </div>
      )}
      {section === "overview" && account && (
        <div className="space-y-6">
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <Stat
              label="Simulated equity"
              value={
                account.id === "us" && account.reconciled_at.startsWith("0001")
                  ? "—"
                  : money(account.equity, currency)
              }
              detail={`${currency} · segregated account`}
            />
            <Stat
              label="P&L realized"
              value={known ? money(account.realized, currency) : "—"}
              detail="Based on recorded fills"
              positive={Number(account.realized) >= 0}
            />
            <Stat
              label="Reserved exposure + positions"
              value={known ? money(account.exposure, currency) : "—"}
              detail="Includes pending orders"
            />
            <Stat
              label="Drawdown atual"
              value={known ? `${Number(account.drawdown).toFixed(2)}%` : "—"}
              detail="Relative to observed peak"
            />
          </div>
          <div className="grid gap-5 xl:grid-cols-[1fr_320px]">
            <Card>
              <CardHeader className="flex-row items-start justify-between">
                <div>
                  <CardTitle>Explore history</CardTitle>
                  <CardDescription>
                    Up to 1,000 recent candles from the selected dataset
                  </CardDescription>
                </div>
                <div className="max-w-48">
                  <DatasetSelect snapshot={snapshot} value={datasetId} onChange={setDataset} />
                </div>
              </CardHeader>
              <CardContent>
                {candles.length ? (
                  <MarketChart candles={candles} />
                ) : (
                  <Empty
                    title="Your chart starts with data"
                    detail="Import an authorized history or query Alpaca Paper in Market data."
                  />
                )}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>Operational status</CardTitle>
                <CardDescription>Control independent of the browser</CardDescription>
              </CardHeader>
              <CardContent className="space-y-5">
                <div className="flex items-center gap-3">
                  {account.halted ? (
                    <ShieldAlert className="text-amber-500" />
                  ) : (
                    <Activity className="text-emerald-500" />
                  )}
                  <div>
                    <p className="text-sm">
                      {account.halted ? "Entries blocked" : "Simulation active"}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {account.reason || "Monitored by the Go worker"}
                    </p>
                  </div>
                </div>
                <div className="space-y-2 border-y py-4 text-xs">
                  <p className="flex justify-between">
                    <span className="text-muted-foreground">Strategies enabled</span>
                    {
                      snapshot.strategies.filter(
                        (s) => s.status === "enabled" && s.payload.account_id === account.id,
                      ).length
                    }
                  </p>
                  <p className="flex justify-between">
                    <span className="text-muted-foreground">Cash</span>
                    {known ? money(account.cash, currency) : "—"}
                  </p>
                  <p className="flex justify-between">
                    <span className="text-muted-foreground">P&L unrealized</span>
                    {known ? money(account.unrealized, currency) : "—"}
                  </p>
                  <p className="text-muted-foreground">Heartbeat: {date(account.heartbeat)}</p>
                </div>
                {controls(account)}
                <p className="text-[11px] leading-relaxed text-muted-foreground">
                  IEX does not represent US consolidated volume. B3 replay uses historical data and
                  internal fills.
                </p>
              </CardContent>
            </Card>
          </div>
          <Card>
            <CardHeader>
              <CardTitle>Latest orders</CardTitle>
            </CardHeader>
            <CardContent>
              <Orders account={account} act={act} />
            </CardContent>
          </Card>
        </div>
      )}
      {section === "market" && (
        <div className="space-y-5">
          <div className="grid gap-5 xl:grid-cols-2">
            <ImportForm snapshot={snapshot} refresh={refresh} setError={setError} />
            <Card>
              <CardHeader>
                <CardTitle>US history · IEX</CardTitle>
                <CardDescription>
                  Uses the Alpaca credentials on the server. Respects the restricted recent window
                  of the basic plan.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <FetchForm configured={snapshot.settings.alpaca_configured} act={act} busy={busy} />
                <p className="mt-5 text-xs leading-relaxed text-muted-foreground">
                  No subscription is purchased automatically. The real-time B3 source is not yet
                  integrated.
                </p>
              </CardContent>
            </Card>
          </div>
          <Card>
            <CardHeader>
              <CardTitle>Available datasets</CardTitle>
              <CardDescription>Preserved with hash, source, license and period</CardDescription>
            </CardHeader>
            <CardContent>
              <GridTable
                headers={["Symbol", "Source", "Candles", "Period", "Quality", "View"]}
                rows={snapshot.datasets.map((ds) =>
                  row(
                    ds.id,
                    <span className="font-mono">{ds.symbol}</span>,
                    ds.source,
                    ds.bars,
                    <span>
                      {date(ds.first)}
                      <br />
                      {date(ds.last)}
                    </span>,
                    ds.problems?.length ? (
                      <Badge variant="outline">{ds.problems.length} gaps</Badge>
                    ) : (
                      <Status value="ready" />
                    ),
                    <Button size="sm" variant="outline" onClick={() => setDataset(ds.id)}>
                      Chart
                    </Button>,
                  ),
                )}
              />
              {candles.length > 0 && (
                <div className="mt-6">
                  <p className="mb-3 text-xs text-muted-foreground">
                    {snapshot.datasets.find((d) => d.id === datasetId)?.symbol} · last{" "}
                    {candles.length} candles
                  </p>
                  <MarketChart candles={candles} />
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}
      {section === "strategies" && (
        <div className="space-y-5">
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              Each configuration is immutable; create a new version to change rules.
            </p>
            <Dialog>
              <DialogTrigger asChild>
                <Button>
                  <Plus />
                  New strategy
                </Button>
              </DialogTrigger>
              <DialogContent className="max-h-[90vh] overflow-auto">
                <DialogHeader>
                  <DialogTitle>New configuration</DialogTitle>
                  <DialogDescription>
                    Starts disabled. Enabling the strategy does not unlock the account.
                  </DialogDescription>
                </DialogHeader>
                <StrategyForm snapshot={snapshot} refresh={refresh} />
              </DialogContent>
            </Dialog>
          </div>
          {snapshot.strategies.length ? (
            <div className="grid gap-5 lg:grid-cols-2 xl:grid-cols-3">
              {snapshot.strategies.map(({ payload: s, status }) => (
                <Card key={s.id}>
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <Badge variant="outline">
                        {s.symbol} · {s.account_id.toUpperCase()}
                      </Badge>
                      <Status value={status} />
                    </div>
                    {s.draft && (
                      <p className="pt-2 text-xs text-amber-500">
                        Draft: review the parameters, then approve it to enable.
                      </p>
                    )}
                    <CardTitle className="pt-3">{s.name}</CardTitle>
                    <CardDescription>
                      {kindName[s.kind]} · version {s.version}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="mb-5 grid grid-cols-2 gap-3 rounded-xl bg-muted/30 p-4 text-xs">
                      <span className="text-muted-foreground">Windows</span>
                      <span>
                        {s.fast} / {s.slow}
                      </span>
                      <span className="text-muted-foreground">Stop / target</span>
                      <span>
                        {s.stop_bps / 100}% / {s.target_bps / 100}%
                      </span>
                      <span className="text-muted-foreground">Timeframe</span>
                      <span>5 minutes · regular session</span>
                    </div>
                    <Button
                      className="w-full"
                      variant={status === "enabled" ? "outline" : "default"}
                      disabled={busy}
                      onClick={() =>
                        void act(`/api/strategies/${s.id}/toggle`, {
                          enabled: status !== "enabled",
                          approve_draft: s.draft && status !== "enabled",
                        })
                      }
                    >
                      {status === "enabled"
                        ? "Disable"
                        : s.draft
                          ? "Approve and enable"
                          : "Enable strategy"}
                    </Button>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <Card>
              <Empty
                title="Which hypothesis do you want to test?"
                detail="Start with moving averages, mean reversion or breakout. Parameters and limits are validated on the server."
              />
            </Card>
          )}
        </div>
      )}
      {section === "backtests" && (
        <div className="space-y-5">
          <Card>
            <CardHeader>
              <CardTitle>Run experiment</CardTitle>
              <CardDescription>
                Durable job processed by the worker. Train and test use a 70% / 30%.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BacktestForm snapshot={snapshot} act={act} busy={busy} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Experiment history</CardTitle>
              <CardDescription>
                Compare net results and sample limits; do not tune parameters on the final test.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <GridTable
                headers={[
                  "Strategy",
                  "Partition",
                  "Status",
                  "Return",
                  "Drawdown",
                  "Trades",
                  "Report",
                ]}
                rows={snapshot.backtests.map(({ id, payload: j, status }) =>
                  row(
                    id,
                    j.strategy.name,
                    j.partition,
                    <Status value={status} />,
                    j.result ? `${Number(j.result.metrics.return_pct).toFixed(2)}%` : "—",
                    j.result ? `${Number(j.result.metrics.max_drawdown_pct).toFixed(2)}%` : "—",
                    j.result?.metrics.trades ?? "—",
                    <div className="flex gap-2">
                      {j.result && (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={async () => {
                            try {
                              setReport(await api<Job>(`/api/backtests/${j.id}`));
                              setReportOpen(true);
                            } catch (e) {
                              setError(String(e));
                            }
                          }}
                        >
                          Open
                        </Button>
                      )}
                      {["queued", "running"].includes(status) && (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => void act(`/api/backtests/${j.id}/cancel`, {})}
                        >
                          Cancel
                        </Button>
                      )}
                      {j.error && <span className="text-red-400">{j.error}</span>}
                    </div>,
                  ),
                )}
              />
            </CardContent>
          </Card>
          <Dialog open={reportOpen} onOpenChange={setReportOpen}>
            <DialogContent className="max-h-[90vh] overflow-auto sm:max-w-4xl">
              <DialogHeader>
                <DialogTitle>{report?.strategy.name} · report</DialogTitle>
                <DialogDescription>
                  Net equity and buy-and-hold benchmark. Partition: {report?.partition}
                </DialogDescription>
              </DialogHeader>
              {report?.result && (
                <>
                  <MarketChart curve={report.result.curve ?? []} />
                  <div className="grid gap-3 sm:grid-cols-3">
                    <Stat
                      label="Simulated costs"
                      value={money(
                        report.result.metrics.costs,
                        snapshot.instruments.find((i) => i.symbol === report.strategy.symbol)
                          ?.currency ?? "USD",
                      )}
                      detail="Fees configured in bps"
                    />
                    <Stat
                      label="Win rate"
                      value={`${report.result.metrics.win_rate.toFixed(1)}%`}
                      detail={`${report.result.metrics.trades} closed trades`}
                    />
                    <Stat
                      label="Exposure"
                      value={String(report.result.metrics.exposure_bars)}
                      detail="Candles with position"
                    />
                  </div>
                  <div className="grid gap-3 sm:grid-cols-3">
                    <Stat
                      label="Sharpe / Sortino"
                      value={`${report.result.metrics.sharpe.toFixed(2)} / ${report.result.metrics.sortino.toFixed(2)}`}
                      detail={`Risk-free ${report.result.metrics.risk_free_annual_pct}% a year`}
                    />
                    <Stat
                      label="Calmar / SQN"
                      value={`${report.result.metrics.calmar.toFixed(2)} / ${report.result.metrics.sqn.toFixed(2)}`}
                      detail="Annualized return over max drawdown"
                    />
                    <Stat
                      label="Buy and hold"
                      value={`${Number(report.result.metrics.benchmark_return_pct).toFixed(2)}%`}
                      detail="No fees, no leverage"
                    />
                  </div>
                  {report.result.robustness && (
                    <div className="space-y-1 rounded-lg bg-muted/40 p-3 text-xs">
                      <p className="font-medium">Overfitting checks</p>
                      <p>
                        Deflated Sharpe {report.result.robustness.deflated_sharpe.toFixed(2)} after{" "}
                        {report.result.robustness.trials} trial(s)
                        {report.result.robustness.low_trades
                          ? " · fewer than 30 trades: weak statistics"
                          : ""}
                      </p>
                      {report.result.robustness.monte_carlo && (
                        <p>
                          Bootstrap return P5 / P50 / P95:{" "}
                          {report.result.robustness.monte_carlo.p5_return_pct.toFixed(2)}% /{" "}
                          {report.result.robustness.monte_carlo.p50_return_pct.toFixed(2)}% /{" "}
                          {report.result.robustness.monte_carlo.p95_return_pct.toFixed(2)}% · loss
                          probability{" "}
                          {(report.result.robustness.monte_carlo.probability_of_loss * 100).toFixed(
                            0,
                          )}
                          %
                        </p>
                      )}
                      {report.result.robustness.cost_stress_return_pct !== undefined && (
                        <p>
                          Return with doubled costs:{" "}
                          {Number(report.result.robustness.cost_stress_return_pct).toFixed(2)}%
                        </p>
                      )}
                    </div>
                  )}
                  {report.result.windows && report.result.windows.length > 0 && (
                    <GridTable
                      headers={["Window", "Bars", "Trades", "Return", "Max drawdown", "Note"]}
                      rows={report.result.windows.map((w) =>
                        row(
                          w.index,
                          w.index,
                          w.bars,
                          w.trades,
                          `${Number(w.return_pct).toFixed(2)}%`,
                          `${Number(w.max_drawdown_pct).toFixed(2)}%`,
                          w.note ?? (w.enough_trades ? "" : "too few trades"),
                        ),
                      )}
                    />
                  )}
                  <div className="space-y-2 text-xs text-muted-foreground">
                    {(report.result.warnings ?? []).map((w) => (
                      <p key={w}>• {w}</p>
                    ))}
                    <p className="break-all">Dataset SHA256: {report.result.dataset_hash}</p>
                  </div>
                  <Button
                    variant="outline"
                    onClick={() => {
                      const blob = new Blob([JSON.stringify(report, null, 2)], {
                        type: "application/json",
                      });
                      const href = URL.createObjectURL(blob);
                      const a = document.createElement("a");
                      a.href = href;
                      a.download = `vukem-trading-backtest-${report.id}.json`;
                      a.click();
                      URL.revokeObjectURL(href);
                    }}
                  >
                    <ArrowDownToLine />
                    Export JSON artifact
                  </Button>
                  <GridTable
                    headers={["Time", "Symbol", "Side", "Quantity", "Price", "Fees"]}
                    rows={(report.result.fills ?? [])
                      .slice(-100)
                      .map((f) =>
                        row(f.id, date(f.time), f.symbol, f.side, f.qty, f.price, f.fees),
                      )}
                  />
                </>
              )}
            </DialogContent>
          </Dialog>
        </div>
      )}
      {section === "orders" && account && (
        <Card>
          <CardHeader>
            <CardTitle>Orders · {account.id.toUpperCase()}</CardTitle>
            <CardDescription>
              Last 100 records. Unknown state keeps the risk reserve.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Orders account={account} act={act} />
          </CardContent>
        </Card>
      )}
      {section === "positions" && account && (
        <div className="space-y-5">
          <Card>
            <CardHeader>
              <CardTitle>Open positions</CardTitle>
              <CardDescription>Current mark and protection set by the entry intent</CardDescription>
            </CardHeader>
            <CardContent>
              <GridTable
                headers={[
                  "Symbol",
                  "Quantity",
                  "Average price",
                  "Mark",
                  "Stop",
                  "Target",
                  "P&L unrealized",
                ]}
                rows={Object.values(account.positions).map((p) =>
                  row(
                    p.symbol,
                    p.symbol,
                    p.qty,
                    money(p.average, currency),
                    money(p.mark, currency),
                    money(p.stop, currency),
                    money(p.target, currency),
                    money((Number(p.mark) - Number(p.average)) * Number(p.qty), currency),
                  ),
                )}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Recorded executions</CardTitle>
              <CardDescription>
                Display values; accounting uses decimals on the server
              </CardDescription>
            </CardHeader>
            <CardContent>
              <GridTable
                headers={["Time", "Symbol", "Side", "Quantity", "Price", "Fees"]}
                rows={[...account.fills]
                  .reverse()
                  .map((f) =>
                    row(
                      f.id,
                      date(f.time),
                      f.symbol,
                      f.side,
                      f.qty,
                      money(f.price, currency),
                      money(f.fees, currency),
                    ),
                  )}
              />
            </CardContent>
          </Card>
        </div>
      )}
      {section === "risk" && account && (
        <div className="grid gap-5 xl:grid-cols-[1fr_360px]">
          <Card>
            <CardHeader>
              <CardTitle>Risk policy · {account.id.toUpperCase()}</CardTitle>
              <CardDescription>
                Block the account to edit. Limits persist across restarts.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <RiskForm key={account.id} account={account} act={act} busy={busy} />
            </CardContent>
          </Card>
          <Card className="border-amber-500/20">
            <CardHeader>
              <ShieldAlert className="mb-2 text-amber-500" />
              <CardTitle>Operational controls</CardTitle>
              <CardDescription>
                Blocking new entries does not mean positions were closed.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              {controls(account)}
              <p className="text-xs leading-relaxed text-muted-foreground">
                In-flight orders need confirmation. Stops and targets are observed by the engine and
                do not guarantee price or protection during downtime. For B3, replay applies rules
                on the historical clock.
              </p>
              <p className="text-xs text-muted-foreground">
                Reconciliation: {date(account.reconciled_at)}
              </p>
            </CardContent>
          </Card>
        </div>
      )}
      {section === "recommendations" && <Recommendations />}
      {section === "audit" && (
        <Card>
          <CardHeader>
            <CardTitle>Audited events</CardTitle>
            <CardDescription>Last 100 persisted events</CardDescription>
          </CardHeader>
          <CardContent>
            <GridTable
              headers={["Time", "Actor", "Action", "Account / record", "Details"]}
              rows={snapshot.audit.map((a) =>
                row(
                  a.id,
                  date(a.time),
                  a.actor,
                  <code>{a.action}</code>,
                  <span className="font-mono text-[10px]">{a.account}</span>,
                  a.action === "simulation.reset" ? (
                    <a
                      className="text-emerald-500 underline"
                      href={`/api/artifacts/${encodeURIComponent(a.details)}`}
                      target="_blank"
                      rel="noreferrer"
                    >
                      Previous snapshot (JSON)
                    </a>
                  ) : (
                    a.details
                  ),
                ),
              )}
            />
          </CardContent>
        </Card>
      )}
      {section === "settings" && (
        <div className="grid gap-5 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Integrations</CardTitle>
              <CardDescription>
                Only configuration presence; secrets are never returned to the browser
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              {[
                ["Login Google", snapshot.settings.google_configured],
                ["Authorized owner", snapshot.settings.owner_configured],
                ["Alpaca Paper", snapshot.settings.alpaca_configured],
              ].map(([label, ready]) => (
                <div key={String(label)} className="flex justify-between border-b pb-3 text-sm">
                  <span>{label}</span>
                  <Status value={ready ? "ready" : "not configured"} />
                </div>
              ))}
              <p className="text-xs text-muted-foreground">
                Configure credentials and callback through the Railway environment. B3 uses
                authorized history import.
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Environment contract</CardTitle>
              <CardDescription>Personal and exclusively simulated scope</CardDescription>
            </CardHeader>
            <CardContent>
              <GridTable
                headers={["Setting", "Value"]}
                rows={[
                  row("mode", "Mode", "Paper only"),
                  row(
                    "live",
                    "Live trading",
                    snapshot.settings.live_trading ? "Enabled" : "Off: no live adapter exists",
                  ),
                  row("feed", "US feed", "IEX"),
                  row("timeframe", "Timeframe", "5 minutes"),
                  row("engine", "Engine", snapshot.settings.engine_version),
                  row("persistence", "Persistence", "PostgreSQL"),
                  row("server-time", "Server time", date(snapshot.server_time)),
                ]}
              />
              <div className="mt-5 flex gap-3 rounded-lg bg-muted/40 p-4 text-xs leading-relaxed text-muted-foreground">
                <CircleHelp className="size-4 shrink-0" />
                Account recovery requires an audited administrative procedure. Changing the
                configured email does not replace an already linked identity automatically.
              </div>
            </CardContent>
          </Card>
          <Card className="xl:col-span-2">
            <CardHeader>
              <CardTitle>Market coverage</CardTitle>
              <CardDescription>
                What each market offers, so no market implies more than it has
              </CardDescription>
            </CardHeader>
            <CardContent>
              <GridTable
                headers={["Market", "Data", "Execution", "Source", "Notes"]}
                rows={snapshot.coverage.map((c) =>
                  row(c.market, c.label, c.data, c.execution, c.source, c.notes),
                )}
              />
            </CardContent>
          </Card>
        </div>
      )}
      <p className="mt-6 flex items-center gap-2 text-[10px] text-muted-foreground">
        <Wifi className="size-3" />
        Snapshot: {date(snapshot.server_time)} · refresh every 10 seconds · {currency}
      </p>
    </>
  );
}
function DatasetSelect({
  snapshot,
  value,
  onChange,
  ...props
}: {
  snapshot: Snapshot;
  value: string;
  onChange: (v: string) => void;
  "aria-label"?: string;
}) {
  return (
    <Select {...props} value={value} onChange={onChange}>
      <option value="">Select history</option>
      {snapshot.datasets.map((d) => (
        <option key={d.id} value={d.id}>
          {d.symbol} · {d.bars} candles · {d.source}
        </option>
      ))}
    </Select>
  );
}
function Confirm({
  title,
  description,
  label,
  disabled,
  action,
}: {
  title: string;
  description: string;
  label: string;
  disabled: boolean;
  action: () => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" disabled={disabled}>
          {label}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={() => setOpen(false)}>
            Back
          </Button>
          <Button
            disabled={disabled}
            onClick={async () => {
              await action();
              setOpen(false);
            }}
          >
            Confirm
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
function Orders({
  account,
  act,
}: {
  account: Account;
  act: (p: string, b: unknown) => Promise<void>;
}) {
  return (
    <GridTable
      headers={["Time", "Symbol", "Side", "Qty / filled", "Limit", "State", "Action"]}
      rows={[...account.orders].reverse().map((o) =>
        row(
          o.id,
          date(o.created_at),
          <span className="font-mono">{o.symbol}</span>,
          <Badge variant="outline">{o.side === "buy" ? "Buy" : "Sell"}</Badge>,
          `${o.qty} / ${o.filled}`,
          money(o.limit, account.currency),
          <span>
            <Status value={o.status} />
            {o.reason && (
              <span className="mt-1 block max-w-48 text-[10px] text-muted-foreground">
                {o.reason}
              </span>
            )}
          </span>,
          ["approved", "dispatching", "unknown", "acknowledged", "partially_filled"].includes(
            o.status,
          ) ? (
            <Button
              size="sm"
              variant="outline"
              onClick={() => void act(`/api/accounts/${account.id}/orders/${o.id}/cancel`, {})}
            >
              Cancel
            </Button>
          ) : (
            "—"
          ),
        ),
      )}
    />
  );
}
function ImportForm({
  snapshot,
  refresh,
  setError,
}: {
  snapshot: Snapshot;
  refresh: () => Promise<void>;
  setError: (v: string) => void;
}) {
  const [symbol, setSymbol] = useState("PETR4");
  const [source, setSource] = useState("");
  const [license, setLicense] = useState("");
  const [csv, setCsv] = useState("");
  const [busy, setBusy] = useState(false);
  const [success, setSuccess] = useState("");
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      const result = await api<{ bars: number }>("/api/datasets/import", {
        symbol,
        source,
        license,
        csv,
      });
      setSuccess(`${result.bars} candles imported with hash and provenance.`);
      setCsv("");
      await refresh();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Import CSV history</CardTitle>
        <CardDescription>
          B3 2026 · timestamp with offset; time,open,high,low,close,volume
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className="grid gap-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Symbol">
              <Select value={symbol} onChange={setSymbol}>
                {snapshot.instruments
                  .filter((i) => i.market === "B3")
                  .map((i) => (
                    <option key={i.symbol}>{i.symbol}</option>
                  ))}
              </Select>
            </Field>
            <Field label="File (up to 1 MiB)">
              <Input
                type="file"
                accept=".csv,text/csv"
                onChange={async (e) => {
                  const f = e.target.files?.[0];
                  if (f) {
                    if (f.size > 1024 * 1024) {
                      setError("Upload limit: 1 MiB");
                      return;
                    }
                    setCsv(await f.text());
                  }
                }}
              />
            </Field>
          </div>
          <Field label="Source / vendor">
            <Input
              required
              maxLength={120}
              value={source}
              onChange={(e) => setSource(e.target.value)}
              placeholder="History vendor"
            />
          </Field>
          <Field label="License / authorized use">
            <Input
              required
              maxLength={300}
              value={license}
              onChange={(e) => setLicense(e.target.value)}
              placeholder="Identify the contract or usage permission"
            />
          </Field>
          <Field label="CSV content">
            <Textarea
              required
              rows={4}
              value={csv}
              onChange={(e) => setCsv(e.target.value)}
              className="font-mono text-xs"
              placeholder="time,open,high,low,close,volume"
            />
          </Field>
          <label className="flex items-start gap-2 text-xs leading-relaxed text-muted-foreground">
            <input type="checkbox" required className="mt-0.5" />I confirm authorization to store
            and process this history. I am not presenting synthetic data as real market data.
          </label>
          <Button disabled={busy}>
            {busy ? <Loader2 className="animate-spin" /> : <Upload />}Import and validate
          </Button>
          {success && (
            <p role="status" className="text-xs text-emerald-500">
              {success}
            </p>
          )}
        </form>
      </CardContent>
    </Card>
  );
}
function FetchForm({
  configured,
  act,
  busy,
}: {
  configured: boolean;
  act: (p: string, b: unknown) => Promise<void>;
  busy: boolean;
}) {
  const [symbol, setSymbol] = useState("AAPL");
  const [days, setDays] = useState(7);
  const needsAlpaca = !symbol.endsWith("USDT");
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void act("/api/datasets/fetch", { symbol, days });
      }}
    >
      <Field label="Symbol">
        <Select value={symbol} onChange={setSymbol}>
          {["AAPL", "NVDA", "TSLA", "BTCUSDT", "ETHUSDT"].map((s) => (
            <option key={s}>{s}</option>
          ))}
        </Select>
      </Field>
      <Field label="Period in days (1–30)">
        <Input
          type="number"
          min={1}
          max={30}
          value={days}
          onChange={(e) => setDays(Number(e.target.value))}
        />
      </Field>
      <Button disabled={(needsAlpaca && !configured) || busy}>
        <RefreshCw />
        {needsAlpaca ? "Fetch IEX candles" : "Fetch Binance candles"}
      </Button>
      {needsAlpaca && !configured && (
        <p className="text-xs text-amber-500">Alpaca Paper is not yet configured on the server.</p>
      )}
    </form>
  );
}
function StrategyForm({ snapshot, refresh }: { snapshot: Snapshot; refresh: () => Promise<void> }) {
  const [s, setS] = useState<Strategy>({
    id: "",
    name: "",
    kind: "ma_cross",
    symbol: "PETR4",
    dataset_id: "",
    account_id: "b3",
    version: 1,
    fast: 9,
    slow: 21,
    threshold: 2,
    stop_bps: 100,
    target_bps: 200,
    enabled: false,
    draft: false,
  });
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const [kinds, setKinds] = useState<KindSpec[]>([]);
  useEffect(() => {
    let active = true;
    api<KindSpec[]>("/api/strategy-kinds")
      .then((k) => active && setKinds(k))
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);
  const spec = kinds.find((k) => k.kind === s.kind);
  const params: ParamSpec[] = spec?.params ?? fallbackParams;
  const change = (key: keyof Strategy, value: string | number) => setS({ ...s, [key]: value });
  const pickKind = (kind: string) => {
    const next = kinds.find((k) => k.kind === kind);
    const defaults = Object.fromEntries((next?.params ?? []).map((p) => [p.name, p.default]));
    setS({ ...s, ...defaults, kind });
  };
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api("/api/strategies", s);
      await refresh();
      setStatus("Configuration created and disabled.");
    } catch (e) {
      setStatus(String(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <form className="grid gap-4" onSubmit={submit}>
      <Field label="Name">
        <Input
          required
          minLength={3}
          maxLength={80}
          value={s.name}
          onChange={(e) => change("name", e.target.value)}
        />
      </Field>
      <Field label="Family">
        <Select value={s.kind} onChange={pickKind}>
          {(kinds.length > 0
            ? kinds.map((k): [string, string] => [k.kind, k.label])
            : Object.entries(kindName)
          ).map(([k, v]) => (
            <option key={k} value={k}>
              {v}
            </option>
          ))}
        </Select>
      </Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Account">
          <Select
            value={s.account_id}
            onChange={(v) =>
              setS({ ...s, account_id: v, symbol: firstSymbol[v] ?? "AAPL", dataset_id: "" })
            }
          >
            <option value="b3">B3 replay</option>
            <option value="us">Alpaca Paper</option>
            <option value="research">Research (backtests only)</option>
          </Select>
        </Field>
        <Field label="Symbol">
          <Select value={s.symbol} onChange={(v) => setS({ ...s, symbol: v, dataset_id: "" })}>
            {snapshot.instruments
              .filter((i) => accountOf(i.market) === s.account_id)
              .map((i) => (
                <option key={i.symbol}>{i.symbol}</option>
              ))}
          </Select>
        </Field>
      </div>
      {s.account_id !== "us" && (
        <Field label="Dataset for replay">
          <DatasetSelect
            snapshot={{
              ...snapshot,
              datasets: snapshot.datasets.filter((d) => d.symbol === s.symbol),
            }}
            value={s.dataset_id}
            onChange={(v) => change("dataset_id", v)}
          />
        </Field>
      )}
      <div className="grid grid-cols-2 gap-3">
        {params.map((p) => (
          <Field key={p.name} label={p.label}>
            <Input
              type="number"
              step={p.step}
              min={p.min}
              max={p.max}
              required
              value={s[p.name as NumericKey]}
              onChange={(e) => change(p.name as NumericKey, Number(e.target.value))}
            />
          </Field>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">
        100 bps = 1%. Signals only on closed candles; immutable version.
      </p>
      <Button disabled={busy}>
        {busy ? <Loader2 className="animate-spin" /> : <Plus />}Create configuration
      </Button>
      {status && (
        <p role="status" className="text-xs">
          {status}
        </p>
      )}
    </form>
  );
}
function BacktestForm({
  snapshot,
  act,
  busy,
}: {
  snapshot: Snapshot;
  act: (p: string, b: unknown) => Promise<void>;
  busy: boolean;
}) {
  const [strategy, setStrategy] = useState("");
  const [dataset, setDataset] = useState("");
  const [partition, setPartition] = useState("test");
  const s = snapshot.strategies.find((s) => s.id === strategy)?.payload;
  return (
    <form
      className="grid gap-4 md:grid-cols-4"
      onSubmit={(e) => {
        e.preventDefault();
        void act("/api/backtests", { strategy_id: strategy, dataset_id: dataset, partition });
      }}
    >
      <Field label="Strategy">
        <Select
          value={strategy}
          onChange={(v) => {
            setStrategy(v);
            setDataset("");
          }}
        >
          <option value="">Select</option>
          {snapshot.strategies.map((s) => (
            <option key={s.id} value={s.id}>
              {s.payload.name} · {s.payload.symbol}
            </option>
          ))}
        </Select>
      </Field>
      <Field label="Dataset">
        <DatasetSelect
          snapshot={{
            ...snapshot,
            datasets: snapshot.datasets.filter((d) => d.symbol === s?.symbol),
          }}
          value={dataset}
          onChange={setDataset}
        />
      </Field>
      <Field label="Time partition">
        <Select value={partition} onChange={setPartition}>
          <option value="train">Train · first 60%</option>
          <option value="test">Test · next 20%</option>
          <option value="validation">Validation · last 20% (use once per version)</option>
          <option value="walkforward">Walk-forward · 4 windows over the full period</option>
          <option value="all">Exploratory · full period</option>
        </Select>
      </Field>
      <div className="flex items-end">
        <Button className="w-full" disabled={!strategy || !dataset || busy}>
          <FlaskConical />
          Enqueue backtest
        </Button>
      </div>
    </form>
  );
}
function RiskForm({
  account,
  act,
  busy,
}: {
  account: Account;
  act: (p: string, b: unknown) => Promise<void>;
  busy: boolean;
}) {
  const [p, setP] = useState<Policy>(account.policy);
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void act(`/api/accounts/${account.id}/policy`, p);
      }}
    >
      {(
        [
          ["risk_bps", "Risk per trade (bps)", 1, 100],
          ["daily_loss_bps", "Max daily loss (bps)", 1, 500],
          ["exposure_bps", "Max exposure (bps)", 100, 10000],
          ["max_positions", "Max positions", 1, 6],
          ["cost_bps", "Simulated costs per side (bps)", 0, 100],
          ["slippage_bps", "Slippage / limit (bps)", 1, 100],
          ["max_age_seconds", "Max price age (seconds)", 60, 900],
          ["stop_streak", "Stop-loss streak before a symbol lock (0 = off)", 0, 10],
          ["streak_lock_mins", "Streak lock length (minutes)", 0, 1440],
          ["cooldown_mins", "Cooldown after any exit (minutes, 0 = off)", 0, 1440],
          ["drawdown_bps", "Equity drawdown halt (bps, 0 = off)", 0, 5000],
        ] as const
      ).map(([key, label, min, max]) => (
        <Field key={key} label={label}>
          <Input
            disabled={!account.halted}
            type="number"
            min={min}
            max={max}
            step={1}
            required
            value={p[key]}
            onChange={(e) => setP({ ...p, [key]: Number(e.target.value) })}
          />
        </Field>
      ))}
      <p className="text-xs leading-relaxed text-muted-foreground sm:col-span-2">
        50 bps = 0.5%. Daily loss considers marked equity and costs. Limits are not guarantees of
        maximum loss: gaps and downtime may exceed them.
      </p>
      <Button disabled={!account.halted || busy} className="sm:col-span-2">
        Save policy and record audit
      </Button>
    </form>
  );
}

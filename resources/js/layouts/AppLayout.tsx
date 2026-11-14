import { Link, router, usePage } from "@inertiajs/react";
import {
  Activity,
  ArrowUpRight,
  BarChart3,
  CandlestickChart,
  FlaskConical,
  Layers,
  ListOrdered,
  LogOut,
  Moon,
  ScrollText,
  Settings2,
  ShieldCheck,
  Sun,
  Terminal,
  TrendingUp,
} from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { CommandPalette } from "@/components/CommandPalette";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { SharedProps } from "@/types";

const navigation = [
  ["/", "Overview", BarChart3],
  ["/market", "Market data", CandlestickChart],
  ["/strategies", "Strategies", Layers],
  ["/backtests", "Lab", FlaskConical],
  ["/recommendations", "Daily ranking", TrendingUp],
  ["/orders", "Orders", ListOrdered],
  ["/positions", "Positions", Activity],
  ["/risk", "Risk control", ShieldCheck],
  ["/audit", "Audit", ScrollText],
  ["/settings", "Settings", Settings2],
] as const;
export default function AppLayout({ children }: { children: ReactNode }) {
  const page = usePage<SharedProps>();
  const user = page.props.auth?.user;
  const commands = useMemo(
    () => [
      ...navigation.map(([href, label]) => ({
        id: href,
        label,
        run: () => router.visit(href),
      })),
      {
        id: "theme",
        label: "Toggle light and dark theme",
        hint: "Theme",
        run: () => document.getElementById("theme-toggle")?.click(),
      },
    ],
    [],
  );
  const [dark, setDark] = useState(
    () => typeof document === "undefined" || document.documentElement.classList.contains("dark"),
  );
  if (!user) return <>{children}</>;
  const current = navigation.find(([href]) => href === page.url.split("?")[0]);
  function toggle() {
    document.documentElement.classList.toggle("dark", !dark);
    localStorage.setItem("theme", dark ? "light" : "dark");
    setDark(!dark);
  }
  return (
    <TooltipProvider>
      <CommandPalette commands={commands} />
      <SidebarProvider>
        <Sidebar variant="inset">
          <SidebarHeader className="p-5 pb-7">
            <Link href="/" className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded-xl bg-emerald-400 text-emerald-950">
                <Terminal className="size-5" />
              </div>
              <div>
                <span className="text-xl font-semibold tracking-tight">
                  vukem-trading<span className="text-emerald-400">.</span>
                </span>
                <p className="text-[10px] uppercase tracking-[.2em] text-muted-foreground">
                  Quant trading terminal
                </p>
              </div>
            </Link>
          </SidebarHeader>
          <SidebarContent>
            <SidebarGroup>
              <p className="px-3 pb-3 text-[10px] uppercase tracking-widest text-muted-foreground">
                Personal workspace
              </p>
              <SidebarMenu>
                {navigation.map(([href, label, Icon]) => (
                  <SidebarMenuItem key={href}>
                    <SidebarMenuButton asChild isActive={current?.[0] === href} className="h-10">
                      <Link href={href} aria-label={label}>
                        <Icon className="size-4" />
                        <span>{label}</span>
                        {href === "/backtests" && (
                          <span className="ml-auto text-[9px] text-emerald-400">LAB</span>
                        )}
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroup>
            <div className="mx-4 mt-auto rounded-xl border border-emerald-400/15 bg-emerald-400/5 p-4">
              <ShieldCheck className="mb-2 size-5 text-emerald-400" />
              <p className="text-xs font-medium">Simulation environment</p>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                Brazil + US. No orders with real capital.
              </p>
            </div>
          </SidebarContent>
          <SidebarFooter className="p-4">
            <div className="flex items-center gap-3 border-t pt-4">
              <span className="flex size-8 items-center justify-center rounded-full bg-muted text-sm">
                {user.name?.slice(0, 1).toUpperCase()}
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-medium">{user.name}</p>
                <p className="truncate text-[10px] text-muted-foreground">Owner account</p>
              </div>
              <Link href="/logout" method="post" as="button" aria-label="Sign out">
                <LogOut className="size-4 text-muted-foreground" />
              </Link>
            </div>
          </SidebarFooter>
        </Sidebar>
        <SidebarInset>
          <header className="flex h-16 items-center gap-3 border-b px-5 md:px-8">
            <SidebarTrigger />
            <span className="text-muted-foreground">/</span>
            <span className="text-sm">{current?.[1] ?? "Terminal"}</span>
            <div className="ml-auto flex items-center gap-3">
              <Badge variant="outline" className="border-emerald-400/30 text-emerald-500">
                PAPER ONLY
              </Badge>
              <Button
                id="theme-toggle"
                variant="ghost"
                size="icon"
                onClick={toggle}
                aria-label="Toggle theme"
              >
                {dark ? <Sun /> : <Moon />}
              </Button>
              <a
                className="hidden text-xs text-muted-foreground md:inline-flex"
                href="https://github.com/AlexGalhardo/vukem-trading"
                target="_blank"
                rel="noreferrer"
              >
                Project <ArrowUpRight className="ml-1 size-3" />
              </a>
            </div>
          </header>
          <main className="mx-auto w-full max-w-[1600px] p-5 md:p-8">{children}</main>
          <footer className="mt-auto flex flex-wrap justify-between gap-2 border-t px-8 py-4 text-[10px] text-muted-foreground">
            <span>VUKEM-TRADING / RESEARCH • SIMULATE • OBSERVE</span>
            <span>
              Simulated results do not guarantee future performance.{" "}
              <a
                href="https://www.tradingview.com/"
                target="_blank"
                rel="noreferrer"
                className="underline"
              >
                TradingView Lightweight Charts™ · © 2025 TradingView, Inc.
              </a>
            </span>
          </footer>
        </SidebarInset>
      </SidebarProvider>
    </TooltipProvider>
  );
}

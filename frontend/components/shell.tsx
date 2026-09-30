"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { api, type User } from "@/lib/api";
import { navItems, roleLabel } from "@/lib/roles";
import { cx, Wordmark } from "./ui";
import { ThemeToggle } from "./theme-toggle";

function isActive(pathname: string, href: string): boolean {
  if (href === "/roteiros/novo") return pathname === href;
  return pathname === href || pathname.startsWith(`${href}/`);
}

function NavList({ user, onNavigate }: { user: User; onNavigate?: () => void }) {
  const pathname = usePathname();
  const items = navItems.filter((i) => i.roles.includes(user.role));
  return (
    <ul className="flex flex-col gap-0.5">
      {items.map((item) => {
        const active = isActive(pathname, item.href);
        return (
          <li key={item.href}>
            <Link
              href={item.href}
              onClick={onNavigate}
              aria-current={active ? "page" : undefined}
              className={cx(
                "group flex items-center gap-3 rounded-lg px-3 py-2.5 text-[15px] font-medium transition-colors",
                active ? "bg-placa-soft text-placa-ink" : "text-ink-2 hover:bg-surface-2 hover:text-ink",
              )}
            >
              <span
                aria-hidden
                className={cx(
                  "h-2.5 w-2.5 shrink-0 rounded-full border-2 transition-colors",
                  active ? "border-placa bg-placa" : "border-line-strong group-hover:border-ink-3",
                )}
              />
              {item.label}
            </Link>
          </li>
        );
      })}
    </ul>
  );
}

function UserBlock({ user }: { user: User }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  async function logout() {
    setBusy(true);
    try {
      await api.logout();
    } catch {
      // The cookie is cleared server-side either way; go to login.
    }
    router.replace("/login");
    router.refresh();
  }
  return (
    <div className="flex flex-col gap-3 border-t border-line pt-4">
      <div className="min-w-0">
        <p className="truncate font-semibold">{user.name}</p>
        <p className="truncate text-sm text-ink-3">{roleLabel[user.role]}</p>
      </div>
      <div className="flex items-center gap-2">
        <ThemeToggle />
        <button
          type="button"
          onClick={logout}
          disabled={busy}
          className="h-9 flex-1 rounded-lg border border-line-strong px-3 text-sm font-semibold text-ink-2 hover:bg-surface-2 hover:text-ink disabled:opacity-60"
        >
          Sair
        </button>
      </div>
    </div>
  );
}

export function Shell({ user, children }: { user: User; children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  return (
    <div className="min-h-dvh lg:grid lg:grid-cols-[248px_1fr]">
      <a
        href="#conteudo"
        className="sr-only focus:not-sr-only focus:fixed focus:left-3 focus:top-3 focus:z-50 focus:rounded-md focus:bg-surface focus:px-3 focus:py-2"
      >
        Pular para o conteúdo
      </a>

      {/* Desktop rail: the column paints the full page height; the rail
          inside stays pinned to the viewport while the page scrolls. */}
      <div className="hidden border-r border-line bg-surface lg:block">
      <aside className="sticky top-0 flex h-dvh flex-col gap-6 px-4 py-5">
        <Link href="/" aria-label="Terminus, início" className="self-start">
          <Wordmark />
        </Link>
        <nav aria-label="Principal" className="flex-1 overflow-y-auto">
          <NavList user={user} />
        </nav>
        <UserBlock user={user} />
      </aside>
      </div>

      {/* Phone / tablet bar */}
      <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-line bg-surface/95 px-4 backdrop-blur lg:hidden">
        <Link href="/" aria-label="Terminus, início">
          <Wordmark size="sm" />
        </Link>
        <button
          type="button"
          aria-expanded={open}
          aria-controls="menu-movel"
          onClick={() => setOpen((v) => !v)}
          className="flex h-11 items-center gap-2 rounded-lg px-3 text-sm font-semibold text-ink-2 hover:bg-surface-2"
        >
          <svg aria-hidden width="18" height="18" viewBox="0 0 18 18">
            {open ? (
              <path d="M4 4l10 10M14 4L4 14" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            ) : (
              <path d="M2 5h14M2 9h14M2 13h14" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            )}
          </svg>
          Menu
        </button>
      </header>
      {open && (
        <div id="menu-movel" className="fixed inset-x-0 top-14 bottom-0 z-20 lg:hidden">
          <button
            type="button"
            aria-label="Fechar menu"
            className="absolute inset-0 bg-ink/30"
            onClick={() => setOpen(false)}
          />
          <nav
            aria-label="Principal"
            className="rise relative flex max-h-full flex-col gap-4 overflow-y-auto border-b border-line bg-surface px-4 pb-5 pt-3"
          >
            <NavList user={user} onNavigate={() => setOpen(false)} />
            <UserBlock user={user} />
          </nav>
        </div>
      )}

      <main id="conteudo" key={pathname} className="min-w-0 px-4 py-6 sm:px-6 lg:px-10 lg:py-9">
        <div className="mx-auto max-w-6xl">{children}</div>
      </main>
    </div>
  );
}

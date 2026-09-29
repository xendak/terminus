import { redirect } from "next/navigation";
import { Shell } from "@/components/shell";
import { SessionProvider } from "@/components/session-context";
import { Wordmark } from "@/components/ui";
import { getSession } from "@/lib/session";

export default async function AppLayout({ children }: LayoutProps<"/">) {
  const s = await getSession();
  if (s.kind === "anonymous") redirect("/login");
  if (s.kind === "unavailable") {
    return (
      <main className="mx-auto flex min-h-dvh max-w-md flex-col items-start justify-center gap-4 px-6">
        <Wordmark />
        <h1 className="display text-2xl font-bold">Servidor indisponível</h1>
        <p className="text-ink-2">
          O Terminus não conseguiu falar com a API agora. Recarregue a página em alguns instantes.
        </p>
      </main>
    );
  }
  return (
    <SessionProvider user={s.me.user}>
      <Shell user={s.me.user}>{children}</Shell>
    </SessionProvider>
  );
}

import { redirect } from "next/navigation";
import { getSession } from "@/lib/session";
import { homeFor } from "@/lib/roles";

export default async function Home() {
  const s = await getSession();
  if (s.kind === "ok") redirect(homeFor(s.me.user.role));
  redirect("/login");
}

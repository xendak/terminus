import type { Metadata } from "next";
import { RouteScreen } from "@/components/route/route-screen";

export const metadata: Metadata = { title: "Roteiro" };

export default async function Page({ params }: PageProps<"/roteiros/[id]">) {
  const { id } = await params;
  return <RouteScreen id={id} />;
}

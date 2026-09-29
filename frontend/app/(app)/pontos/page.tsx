import type { Metadata } from "next";
import { Locations } from "./locations";

export const metadata: Metadata = { title: "Pontos" };

export default function Page() {
  return <Locations />;
}

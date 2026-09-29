import type { Metadata } from "next";
import { Drivers } from "./drivers";

export const metadata: Metadata = { title: "Motoristas" };

export default function Page() {
  return <Drivers />;
}

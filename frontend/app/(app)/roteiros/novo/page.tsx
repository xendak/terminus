import type { Metadata } from "next";
import { Builder } from "./builder";

export const metadata: Metadata = { title: "Novo roteiro" };

export default function Page() {
  return <Builder />;
}

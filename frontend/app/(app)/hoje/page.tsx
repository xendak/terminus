import type { Metadata } from "next";
import { Today } from "./today";

export const metadata: Metadata = { title: "Meu roteiro de hoje" };

export default function Page() {
  return <Today />;
}

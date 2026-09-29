import type { Metadata } from "next";
import { Params } from "./params";

export const metadata: Metadata = { title: "Parâmetros" };

export default function Page() {
  return <Params />;
}

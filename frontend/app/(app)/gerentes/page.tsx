import type { Metadata } from "next";
import { Managers } from "./managers";

export const metadata: Metadata = { title: "Gerentes" };

export default function Page() {
  return <Managers />;
}

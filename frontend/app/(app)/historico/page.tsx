import type { Metadata } from "next";
import { Suspense } from "react";
import { History } from "./history";

export const metadata: Metadata = { title: "Histórico" };

export default function Page() {
  return (
    <Suspense>
      <History />
    </Suspense>
  );
}

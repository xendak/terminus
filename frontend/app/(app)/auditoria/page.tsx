import type { Metadata } from "next";
import { Suspense } from "react";
import { Audit } from "./audit";

export const metadata: Metadata = { title: "Auditoria" };

export default function Page() {
  return (
    <Suspense>
      <Audit />
    </Suspense>
  );
}

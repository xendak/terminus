"use client";

import type { ManagerOption } from "@/lib/api";
import { Select } from "./ui";

/** Staff filter by team (the drivers of one manager); "" = every team. */
export function TeamSelect({
  id,
  value,
  managers,
  onChange,
  name,
}: {
  id: string;
  value: string;
  managers: ManagerOption[];
  onChange?: (value: string) => void;
  name?: string;
}) {
  const known = managers.some((m) => m.id === value);
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-sm font-medium text-ink-2">
        Equipe
      </label>
      <Select
        id={id}
        name={name}
        {...(onChange ? { value, onChange: (e) => onChange(e.target.value) } : { defaultValue: value })}
        className="min-w-52"
      >
        <option value="">Todas as equipes</option>
        {!known && value && <option value={value}>Equipe selecionada</option>}
        {managers
          .filter((m) => m.active || (m.team_size ?? 0) > 0 || m.id === value)
          .map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
              {typeof m.team_size === "number" ? ` (${m.team_size})` : ""}
              {m.active ? "" : " — inativo"}
            </option>
          ))}
      </Select>
    </div>
  );
}

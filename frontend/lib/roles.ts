import type { Role } from "./api";

export interface NavItem {
  href: string;
  label: string;
  roles: Role[];
}

// Mirrors the role matrix in docs/spec/operations.md; the backend stays
// the authority, this only hides what a role cannot use.
export const navItems: NavItem[] = [
  { href: "/hoje", label: "Meu roteiro de hoje", roles: ["driver"] },
  { href: "/painel", label: "Painel", roles: ["admin", "manager", "driver"] },
  { href: "/historico", label: "Histórico", roles: ["admin", "manager", "driver"] },
  { href: "/roteiros/novo", label: "Novo roteiro", roles: ["admin", "manager"] },
  { href: "/motoristas", label: "Motoristas", roles: ["admin", "manager"] },
  { href: "/pontos", label: "Pontos", roles: ["admin", "manager"] },
  { href: "/gerentes", label: "Gerentes", roles: ["admin"] },
  { href: "/parametros", label: "Parâmetros", roles: ["admin", "manager"] },
  { href: "/auditoria", label: "Auditoria", roles: ["admin"] },
];

export const roleLabel: Record<Role, string> = {
  admin: "Administrador",
  manager: "Gestor",
  driver: "Motorista",
};

export function homeFor(role: Role): string {
  return role === "driver" ? "/hoje" : "/painel";
}

export const isStaff = (role: Role) => role === "admin" || role === "manager";

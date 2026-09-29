import { ApiError } from "./api";

// Backend messages are English and developer-facing; the UI speaks
// pt-BR. Translate by (field, reason) first, then by status.

const reasons: Record<string, string> = {
  required: "Campo obrigatório.",
  "must contain @": "Informe um e-mail válido.",
  "masked value; type the full document": "Digite o CPF completo ou deixe em branco para manter o atual.",
  "earlier than the departure from the previous stop":
    "Horário fora de ordem: a chegada não pode ser antes da saída do ponto anterior.",
  "later than the arrival at the next stop": "Horário fora de ordem: a saída não pode ser depois da chegada ao próximo ponto.",
  "must have at least 8 characters": "Use pelo menos 8 caracteres.",
  "must be positive": "Informe um valor maior que zero.",
  "must not be negative": "Valores negativos não são aceitos.",
  "a route needs at least 2 stops": "O roteiro precisa de pelo menos 2 pontos: a partida e uma parada.",
  "not a driver": "Escolha um motorista.",
  "window start after end": "A data inicial precisa ser anterior à final.",
  "already recorded; use UpdateStopTimes": "Esse horário já foi registrado. Peça a correção ao gestor.",
  "record arrival before departure": "Registre a chegada antes da saída.",
  "not active": "O roteiro não está em andamento.",
  "not closed": "O roteiro não está encerrado.",
  "cannot move up": "Esse ponto já é o primeiro.",
  "cannot move down": "Esse ponto já é o último.",
  "route is full": "O roteiro atingiu o limite de pontos.",
  "distance is set on active routes or at close": "A distância é informada com o roteiro em andamento.",
  "invalid distance": "Informe a distância em km, maior que zero.",
  "invalid km per liter": "Informe o consumo em km/l, maior que zero.",
  "invalid parameter": "Valor inválido para esse parâmetro.",
  "must be date + HH:MM": "Informe data e hora.",
};

function byMessage(msg: string): string | null {
  if (msg.includes("driver already has a route on this date"))
    return "Esse motorista já tem um roteiro nessa data.";
  if (msg.includes("email already registered")) return "Esse e-mail já está cadastrado.";
  if (msg.includes("departure before arrival")) return "A saída não pode ser antes da chegada.";
  if (msg.includes("route is closed")) return "O roteiro está encerrado e não aceita alterações.";
  if (msg.includes("must be YYYY-MM-DD")) return "Data inválida.";
  return null;
}

/** Human pt-BR message for any thrown value. */
export function describeError(err: unknown): string {
  if (!(err instanceof ApiError)) return "Algo deu errado. Tente de novo.";
  if (err.reason && reasons[err.reason]) return reasons[err.reason];
  if (err.reason?.startsWith("must be between")) return "Posição fora do roteiro.";
  const seq = err.reason?.match(/^record the departure from stop (\d+) first$/);
  if (seq) return `Registre antes a saída da parada ${Number(seq[1]) - 1}.`;
  const known = byMessage(err.message);
  if (known) return known;
  switch (err.status) {
    case 0:
      return "Sem conexão com o servidor. Verifique a internet e tente de novo.";
    case 400:
      return "Algum campo está em formato inválido. Revise e tente de novo.";
    case 401:
      return "Sua sessão expirou. Entre de novo.";
    case 403:
      return "Seu perfil não tem acesso a esta ação.";
    case 404:
      return "Registro não encontrado.";
    case 409:
      return "Conflito com um registro existente.";
    case 422:
      return "Revise os valores informados.";
    default:
      return "O servidor não conseguiu concluir. Tente de novo em instantes.";
  }
}

/** Field errors keyed by input name, for inline display. */
export function fieldErrors(err: unknown): Record<string, string> {
  if (!(err instanceof ApiError)) return { form: describeError(err) };
  if (err.message.includes("email already registered")) return { email: describeError(err) };
  if (err.field) return { [err.field]: describeError(err) };
  return { form: describeError(err) };
}

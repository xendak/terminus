# Especificação do Produto — StopTime (título de trabalho)

**Disciplina:** Engenharia de Software II — 2º Trabalho Avaliativo
**Entrega:** 1ª parte — Projeto Preliminar (Especificação)
**Produto:** MVP — Sistema de Monitoramento de Tempo Parado em Roteiros

O entregável é o próprio repositório. Este documento é a especificação da 1ª
parte; a 2ª parte (implementação do MVP) está planejada em `plans/` e detalhada
na especificação técnica em inglês, em `docs/spec/`.

## Política de nomes deste documento

Identificadores do sistema — tabelas, colunas, classes, atributos, operações,
telas e valores de `role` — aparecem exatamente como existem no código e na
especificação técnica, em inglês (por exemplo, `app_user`, `route_stop`,
`arrival_at`, `CreateRoute`). Nomes traduzidos descreveriam um sistema que não
existe. Toda a prosa explicativa deste documento está em português. Os atores
aparecem como personas (Motorista, Gerente, Administrador), ligadas ao valor
real do campo `role` quando relevante.

## 1. Apresentação do produto

Empresas de logística e entrega urbana precisam saber onde e por quanto tempo
seus profissionais de campo ficam parados durante o roteiro diário. Hoje esse
tempo é invisível: não há registro confiável de quanto tempo o entregador, o
motorista ou o transportador permanece em cada ponto do trajeto, o que impede
identificar gargalos, renegociar prazos com clientes e calcular o custo real de
cada rota.

O StopTime é um MVP web que mede esse tempo parado. O produto:

1. Registra motoristas/motoboys, gerentes/coordenadores, locais (pontos) e
   roteiros diários.
2. Coleta, em cada parada do roteiro, o endereço e os horários de chegada e
   saída.
3. Calcula automaticamente o tempo parado por parada e o total do roteiro.
4. Apresenta um dashboard com gráficos de tempo parado por dia, por mês e por
   período, sempre associado aos pontos do roteiro.
5. Calcula indicadores de custo do trajeto (combustível, rendimento km/l,
   custo por km) e o percentual da jornada padrão de 8 h consumido parado.

## 2. Escopo

### 2.1 Dentro do escopo

- Cadastro de motoristas/motoboys, gerentes/coordenadores, locais e roteiros.
- Coleta de dados dos pontos do roteiro (endereço, data/hora de chegada e
  saída).
- Cálculo do tempo parado por ponto e do tempo total parado por roteiro.
- Histórico de pontos e tempos parados por período, com endereços.
- Dashboard com gráficos por dia, por mês e por período.
- Parametrização de custos e da jornada padrão de trabalho (8 h/dia).

### 2.2 Fora do escopo

- Roteirização automática ou otimização de rotas.
- Integração com sistemas de folha de pagamento ou ERP.
- Rastreamento em tempo real com telemetria embarcada no veículo.
- Aplicativo nativo publicado em lojas de aplicativos.

## 3. Requisitos

### 3.1 Regras de negócio

| ID  | Regra                                                                                                                                                                                          |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| RN01 | O ponto de partida não conta tempo parado: o cronômetro de parada só é considerado a partir do segundo ponto do roteiro (`stop_order = 1` nunca acumula).                                      |
| RN02 | O tempo parado em um ponto é a diferença entre o horário de saída (`departure_at`) e o horário de chegada (`arrival_at`) naquele ponto.                                                        |
| RN03 | O tempo total parado do roteiro é a soma dos tempos parados de todos os pontos, exceto o ponto de partida.                                                                                     |
| RN04 | A jornada padrão de trabalho é de 8 horas por dia e serve de base percentual para os indicadores de tempo parado (`journey_percent`).                                                            |
| RN05 | Cada roteiro pertence a um único motorista/motoboy e a uma única data (restrição `UNIQUE (driver_user_id, route_date)`).                                                                        |
| RN06 | Os pontos de um roteiro possuem ordem sequencial (`stop_order` 1, 2, 3, 4...) que define o trajeto do dia.                                                                                     |
| RN07 | O custo do trajeto é calculado a partir do valor do combustível, do rendimento km/litro do veículo e da distância percorrida (`distance_km`).                                                    |

### 3.2 Requisitos funcionais

| ID   | Requisito                                                                                          | Prioridade |
| ---- | -------------------------------------------------------------------------------------------------- | ---------- |
| RF01 | Cadastrar dados do motorista/motoboy (nome, telefone, documento, veículo).                         | Alta       |
| RF02 | Cadastrar dados do gerente/coordenador (nome, telefone, e-mail).                                    | Alta       |
| RF03 | Cadastrar pontos com endereço e coordenadas.                                                        | Alta       |
| RF04 | Montar o roteiro diário associando pontos em ordem sequencial a um motorista e a uma data.          | Alta       |
| RF05 | Registrar chegada e saída em cada ponto (data/hora) para coleta do tempo parado.                     | Alta       |
| RF06 | Calcular automaticamente o tempo parado por ponto e o total do roteiro.                             | Alta       |
| RF07 | Exibir histórico de pontos e tempos parados por período, com endereços.                             | Alta       |
| RF08 | Exibir dashboard com gráficos de tempo parado por dia, por mês e por período.                       | Alta       |
| RF09 | Parametrizar custos: valor do combustível, km/litro do veículo, custo por km percorrido.             | Média      |
| RF10 | Parametrizar regras de cálculo do tempo parado e a jornada padrão (8 h/dia).                          | Média      |
| RF11 | Calcular o custo estimado do roteiro a partir dos parâmetros e da distância percorrida.               | Média      |
| RF12 | Exportar relatórios do período consultado.                                                           | Baixa      |

### 3.3 Requisitos não funcionais

| ID    | Requisito                                                                                          |
| ----- | --------------------------------------------------------------------------------------------------- |
| RNF01 | Persistência dos dados em banco de dados, garantindo o histórico completo dos roteiros.             |
| RNF02 | Interface web responsiva, utilizável em desktop e em dispositivos móveis pelo entregador.           |
| RNF03 | Tempo de resposta do dashboard inferior a 3 segundos para consultas de até 12 meses.                |
| RNF04 | Controle de acesso por perfil: motorista/motoboy, gerente/coordenador e administrador.               |
| RNF05 | Registro de auditoria das alterações em pontos e horários.                                           |
| RNF06 | Aderência à LGPD no tratamento dos dados pessoais dos profissionais de campo.                        |

## 4. Atores

| Ator (persona)          | Valor de `role` | O que faz no sistema                                                                                       |
| ----------------------- | -------------- | ----------------------------------------------------------------------------------------------------------- |
| Motorista / Motoboy     | `driver`       | Executa o próprio roteiro do dia e registra chegada e saída em cada parada; informa a distância percorrida. |
| Gerente / Coordenador  | `manager`      | Cadastra motoristas e locais, monta roteiros, corrige horários, consulta dashboards e histórico, ajusta parâmetros. |
| Administrador (dono)   | `admin`        | Tudo o que o gerente faz, mais contas de gerente, consulta de auditoria e reabertura de roteiros.           |

## 5. Casos de uso

### 5.1 Diagrama de casos de uso

![Diagrama de casos de uso](especificacao/diagrams/casos-de-uso.svg)

Fonte: [`especificacao/diagrams/casos-de-uso.puml`](especificacao/diagrams/casos-de-uso.puml) (PlantUML).

### 5.2 Descrição dos casos de uso

**UC01 — Autenticar-se.**
Ator: todos. Pré-condição: conta ativa (`active = true`).
Fluxo principal: 1) o usuário informa `email` e `password` na tela de login;
2) o sistema valida a senha (bcrypt); 3) o sistema emite a sessão (cookie
assinado com HMAC, validade de 12 h); 4) o redirecionamento segue o `role`:
`driver` vai para a tela Route tracker, os demais para o Dashboard.
Fluxo alternativo: credenciais inválidas → mensagem de erro, nenhuma sessão.
Pós-condição: sessão ativa. Operações: `Login`.

**UC02 — Cadastrar motorista.**
Ator: Gerente, Administrador. Pré-condição: sessão autenticada com `role`
`manager` ou `admin`.
Fluxo principal: 1) o gerente informa `name`, `email`, `password`, `phone` e,
opcionalmente, `document`, `vehicle_name`, `vehicle_plate` e `km_per_l`;
2) o sistema cria o `app_user` com `role = driver` e o `driver_profile` na
mesma transação; 3) o motorista passa a aparecer no diretório.
Fluxo alternativo: e-mail já cadastrado → `ErrDuplicateEmail`, erro exibido no
campo. Pós-condição: motorista cadastrado. Operações: `CreateDriver`,
`ListDrivers`, `UpdateDriver`.

**UC03 — Cadastrar gerente.**
Ator: Administrador. Mesmo formato de UC02, sem perfil de veículo: cria um
`app_user` com `role = manager`. Operações: `CreateManager`, `ListManagers`.

**UC04 — Cadastrar local (ponto).**
Ator: Gerente, Administrador.
Fluxo principal: 1) informa `label`, `address` e, opcionalmente, `latitude` e
`longitude`; 2) o sistema persiste em `location`.
Fluxo alternativo: endereço vazio → erro de campo.
Pós-condição: local disponível para montagem de roteiros. Operações:
`CreateLocation`, `UpdateLocation`, `ListLocations`.

**UC05 — Montar roteiro diário.**
Ator: Gerente (Administrador substitui). Pré-condição: motorista e locais
cadastrados.
Fluxo principal: 1) o gerente seleciona `driver_user_id` e `route_date`;
2) o sistema verifica RN05 (um único roteiro por motorista e data); 3) o gerente
adiciona locais na ordem de visita e o sistema numera `stop_order` de 1 a n
(RN06); 4) ao salvar, o roteiro é criado com `status = draft`; o primeiro ponto
é a partida e nunca acumula tempo parado (RN01).
Fluxos alternativos: 2a) já existe roteiro para o motorista na data → o sistema
exibe o existente (RN05); 3a) menos de duas paradas → erro de campo.
Pós-condição: roteiro `draft` com paradas ordenadas. Operações: `CreateRoute`,
`AddStop`, `RemoveStop`, `ReorderStops`.

**UC06 — Executar roteiro e registrar horários.**
Ator: Motorista. Pré-condição: existe roteiro próprio na data, `status`
`active` (após `StartRoute`).
Fluxo principal: 1) o motorista abre a tela Route tracker e o sistema mostra o
roteiro do dia; 2) em cada parada a partir da segunda (RN01), ele marca a
chegada e o sistema grava `arrival_at`; 3) ao sair, marca a saída e o sistema
grava `departure_at`; 4) o sistema calcula `stop_seconds` por parada (RN02) e o
total do roteiro (RN03).
Fluxos alternativos: 1a) sem roteiro na data → estado vazio com orientação de
contatar o coordenador; 3a) registro em atraso → entrada manual de data/hora em
campo ainda vazio (correções de valor já gravado passam por UC07).
Pós-condição: paradas contadas com chegada e saída; totais visíveis.
Operações: `StartRoute`, `RecordArrival`, `RecordDeparture`, `GetRoute`.

**UC07 — Corrigir horários registrados.**
Ator: Gerente (Administrador). Pré-condição: o roteiro existe e não está
`closed`.
Fluxo principal: 1) o gerente abre o detalhe do roteiro na tela History;
2) edita `arrival_at` ou `departure_at`; 3) o sistema valida RN02
(`departure_at >= arrival_at`) e persiste; 4) grava auditoria com valores
antigos e novos (RNF05); 5) os totais recalculam na leitura seguinte.
Fluxos alternativos: 3a) par de horários inválido → `ErrDepartureBeforeArrival`,
sem auditoria; 3b) roteiro `closed` → `ErrRouteClosed` (reabertura é ação do
administrador, auditada).
Pós-condição: correção persistida e rastreável. Operações: `UpdateStopTimes`,
`ListAudit`.

**UC08 — Encerrar roteiro e informar distância.**
Ator: Motorista (Gerente e Administrador substituem). Pré-condição: roteiro
`active` com horários registrados.
Fluxo principal: 1) o motorista informa `distance_km` (odômetro ou estimativa);
2) encerra o roteiro; 3) o sistema congela composição e horários
(`status = closed`) e apresenta o custo estimado (RN07) e o percentual da
jornada (RN04, padrão de 8 h/dia).
Pós-condição: roteiro somente leitura. Operações: `SetRouteDistance`,
`CloseRoute`.

**UC09 — Consultar dashboard.**
Ator: Gerente e Administrador (o Motorista vê apenas os próprios dados).
Fluxo principal: 1) escolhe um período (predefinido ou personalizado); 2) o
sistema agrega em SQL os minutos parados por dia (`GetDashboardByDay`), por mês
(`GetDashboardByMonth`) e o total do período com ranking por motorista
(`GetDashboardByPeriod`); 3) os gráficos recebem apenas séries agregadas;
4) o percentual da jornada aparece por faixa (RN04).
RNF03: resposta inferior a 3 s para janelas de até 12 meses, verificada com
dados sintéticos de 12 meses.
Pós-condição: os três recortes pedidos (dia, mês, período) visíveis.

**UC10 — Consultar histórico e exportar.**
Ator: Gerente e Administrador (o Motorista vê apenas os próprios dados).
Fluxo principal: 1) filtra por período, motorista e `status`; 2) o sistema
lista os roteiros com totais e custos; 3) o detalhe do roteiro mostra cada
parada com endereço e horários (RF07); 4) exporta o período consultado em CSV
(RF12; UTF-8 com BOM, RFC 4180).
Pós-condição: relatório consultado/exportado. Operações: `ListRoutes`,
`GetRoute`, `ExportPeriodCSV`.

**UC11 — Gerenciar parâmetros.**
Ator: Gerente, Administrador.
Fluxo principal: 1) abre a tela Parameters e o sistema lista `fuel_price_brl`,
`cost_per_km_brl`, `default_km_per_l`, `standard_journey_hours` (padrão 8) e
`min_stop_minutes`; 2) edita um valor; 3) o sistema valida e persiste, gravando
auditoria (RNF05); 4) as leituras seguintes recalculam custo (RN07) e
percentual da jornada (RN04) com o novo valor.
Pós-condição: parâmetro alterado sem mudança de código (critério de aceitação).
Operações: `GetParams`, `UpdateParam`.

**UC12 — Consultar auditoria.**
Ator: Administrador.
Fluxo principal: 1) filtra por `entity` e período; 2) o sistema lista as
entradas de `audit_log` com autor (`actor_user_id`), ação e valores
antigos/novos.
Pós-condição: alterações em pontos e horários rastreáveis (RNF05). Operações:
`ListAudit`.

## 6. Diagramas de robustez

O diagrama de robustez (análise de robustez, Jacobson/ICONIX) reconta o fluxo
de um caso de uso com quatro estereótipos: **ator** (boneco), **boundary**
(interface com o usuário, círculo com traço vertical), **control** (lógica da
aplicação, círculo com seta) e **entity** (dados persistentes, círculo com traço
horizontal). Ele valida, antes do detalhamento, que cada passo do caso de uso
tem um lugar claro no desenho. Nesta especificação, os elementos seguem o
mesmo vínculo em toda parte: boundaries são telas (de `docs/spec/screens.md`),
controls são operações (de `docs/spec/operations.md`) e entities são tabelas (de
`docs/spec/data-model.md`).

### UC05 — Montar roteiro diário

![Robustez UC05](especificacao/diagrams/robustez-uc05-montar-roteiro.svg)

A tela Route builder aciona `CreateRoute`, que aplica RN05 (unicidade
motorista+data) e RN06 (numeração sequencial) antes de gravar `route`; as
edições de composição (`AddStop`, `ReorderStops`, `RemoveStop`) gravam
`route_stop` consultando `location`.

Fonte: [`robustez-uc05-montar-roteiro.puml`](especificacao/diagrams/robustez-uc05-montar-roteiro.puml)

### UC06 — Executar roteiro e registrar horários

![Robustez UC06](especificacao/diagrams/robustez-uc06-registrar-tempos.svg)

A tela Route tracker aciona `StartRoute`, `RecordArrival` e `RecordDeparture`,
que gravam `arrival_at` e `departure_at` em `route_stop`; o cálculo de
`stop_seconds` (RN01+RN02) e o total do roteiro (RN03) completam o fluxo até
`route`.

Fonte: [`robustez-uc06-registrar-tempos.puml`](especificacao/diagrams/robustez-uc06-registrar-tempos.puml)

### UC07 — Corrigir horários registrados

![Robustez UC07](especificacao/diagrams/robustez-uc07-corrigir-tempos.svg)

Na tela History (detalhe do roteiro), `UpdateStopTimes` altera `route_stop` e
grava a auditoria em `audit_log` na mesma transação (RNF05).

Fonte: [`robustez-uc07-corrigir-tempos.puml`](especificacao/diagrams/robustez-uc07-corrigir-tempos.puml)

### UC09 — Consultar dashboard

![Robustez UC09](especificacao/diagrams/robustez-uc09-dashboard.svg)

A tela Dashboard aciona `GetDashboardByDay`, `GetDashboardByMonth` e
`GetDashboardByPeriod`, que agregam `route_stop` e `route` em SQL, leem
`parameter` para o percentual da jornada (RN04) e o custo (RN07), e devolvem
séries agregadas aos gráficos.

Fonte: [`robustez-uc09-dashboard.puml`](especificacao/diagrams/robustez-uc09-dashboard.puml)

## 7. Diagrama de classes conceitual

![Classes conceituais](especificacao/diagrams/classes-conceituais.svg)

Fonte: [`classes-conceituais.puml`](especificacao/diagrams/classes-conceituais.puml)

Atributos precedidos de `/` são derivados: calculados a partir de outros
dados, nunca armazenados (`/stop_seconds` por RN01+RN02,
`/total_stopped_minutes` por RN03, `/journey_percent` por RN04 e
`/estimated_cost` por RN07).

Correspondência entre as classes conceituais e as tabelas de persistência:

| Classe     | Tabelas                                              |
| ---------- | ---------------------------------------------------- |
| User       | `app_user`                                           |
| Driver     | `app_user` (`role = driver`) + `driver_profile`      |
| Manager    | `app_user` (`role = manager`)                        |
| Route      | `route`                                              |
| Stop       | `route_stop`                                         |
| Location   | `location`                                           |
| Parameter  | `parameter`                                          |
| AuditEntry | `audit_log`                                          |

## 8. Diagrama entidade-relacionamento (notação crow's foot)

O diagrama abaixo é renderizado em Mermaid (a notação `erDiagram` do Mermaid é
crow's foot) e espelha o modelo de dados autoritativo, definido em
`docs/spec/data-model.md` e implementado nas migrações SQL de `db/migrations/`.

```mermaid
erDiagram
    APP_USER ||--o| DRIVER_PROFILE : "has (role=driver)"
    APP_USER ||--o{ ROUTE : "drives"
    APP_USER ||--o{ ROUTE : "created_by"
    ROUTE ||--|{ ROUTE_STOP : "has ordered stops"
    LOCATION ||--o{ ROUTE_STOP : "visited as"
    APP_USER ||--o{ AUDIT_LOG : "acted"
    ROUTE }o--|| PARAMETER : "all routes read"

    APP_USER {
        uuid id PK
        text name
        text email UK
        text phone
        text password_hash
        text role "admin|manager|driver"
        bool active
        timestamptz created_at
    }
    DRIVER_PROFILE {
        uuid user_id PK_FK
        text document
        text vehicle_name
        text vehicle_plate
        numeric km_per_l "nullable, overrides default param"
    }
    LOCATION {
        uuid id PK
        text label
        text address
        numeric latitude "nullable"
        numeric longitude "nullable"
        uuid created_by FK
        timestamptz created_at
    }
    ROUTE {
        uuid id PK
        uuid driver_user_id FK
        date route_date "unique with driver"
        numeric distance_km "nullable, manual entry"
        text status "draft|active|closed"
        text note "nullable"
        uuid created_by FK
        timestamptz created_at
    }
    ROUTE_STOP {
        uuid id PK
        uuid route_id FK
        int stop_order "1..n, 1 = departure"
        uuid location_id FK
        timestamptz arrival_at "nullable"
        timestamptz departure_at "nullable"
        int stop_seconds "generated, RN01+RN02"
        text note "nullable"
    }
    PARAMETER {
        text key PK
        numeric value
        text unit
        uuid updated_by FK
        timestamptz updated_at
    }
    AUDIT_LOG {
        uuid id PK
        timestamptz at
        uuid actor_user_id FK
        text entity
        uuid entity_id
        text action
        jsonb old_values
        jsonb new_values
    }
```

Notas: `stop_seconds` é uma coluna gerada no banco (RN01 e RN02 aplicados no
próprio esquema); totais e custo nunca são armazenados, são calculados na
leitura; `schema_migrations` controla as migrações e não aparece no modelo
conceitual.

## 9. Exemplo de referência (dados de validação)

O exemplo do enunciado é a fixture de teste do projeto (semente dourada em
`db/seed/golden.sql`). O ponto 1 é a partida e não acumula tempo parado (RN01).

| Roteiro | Ponto / Endereço            | Tempo parado | Observação                       |
| ------- | --------------------------- | ------------ | -------------------------------- |
| A       | 1 — Seg. Família (partida)  | 0            | Ponto de partida não conta tempo |
| A       | 2 — Rua Peru, 55            | 15 min       | Parada intermediária             |
| A       | 3 — Rua X, 5                | 10 min       | Parada intermediária             |
| A       | 4 — Av. João César          | 50 min       | Ponto final                      |
| B       | 1 — Partida                 | 0            | Não conta tempo                  |
| B       | 2                           | 10 min       |                                  |
| B       | 3                           | 5 min        |                                  |
| B       | 4                           | 26 min       |                                  |
| C       | 1 — Partida                 | 0            | Não conta tempo                  |
| C       | 2                           | 5 min        |                                  |
| C       | 3                           | 10 min       |                                  |
| C       | 4                           | 30 min       |                                  |

Totais esperados, verificados em todas as camadas (SQL, regras de domínio e
dashboard): roteiro A = 75 min; roteiro B = 41 min; roteiro C = 45 min; dia
com os três roteiros = 161 min. Percentual da jornada do roteiro A com o padrão
de 8 h: 75/480 = 15,625%.

## 10. Matriz de rastreabilidade

| UC   | Requisitos       | Regras     | Operações                                                                   |
| ---- | ---------------- | ---------- | --------------------------------------------------------------------------- |
| UC01 | RNF04            |            | `Login`, `Logout`                                                            |
| UC02 | RF01             |            | `CreateDriver`, `UpdateDriver`, `ListDrivers`                                 |
| UC03 | RF02             |            | `CreateManager`, `ListManagers`                                               |
| UC04 | RF03             |            | `CreateLocation`, `UpdateLocation`, `ListLocations`                          |
| UC05 | RF04             | RN01, RN05, RN06 | `CreateRoute`, `AddStop`, `RemoveStop`, `ReorderStops`                 |
| UC06 | RF05, RF06       | RN01, RN02, RN03 | `StartRoute`, `RecordArrival`, `RecordDeparture`, `GetRoute`           |
| UC07 | RF05, RNF05      | RN02       | `UpdateStopTimes`, `ListAudit`                                                |
| UC08 | RF11             | RN04, RN07 | `SetRouteDistance`, `CloseRoute`                                              |
| UC09 | RF08, RNF03      | RN04       | `GetDashboardByDay`, `GetDashboardByMonth`, `GetDashboardByPeriod`            |
| UC10 | RF07, RF12       |            | `ListRoutes`, `GetRoute`, `ExportPeriodCSV`                                  |
| UC11 | RF09, RF10       | RN04, RN07 | `GetParams`, `UpdateParam`                                                    |
| UC12 | RNF05            |            | `ListAudit`                                                                   |

Requisitos não funcionais e onde são atendidos:

| RNF    | Onde é atendido                                                                                                |
| ------ | ---------------------------------------------------------------------------------------------------------------- |
| RNF01  | `docs/spec/data-model.md` (PostgreSQL, histórico completo) e migrações em `db/migrations/`                        |
| RNF02  | Telas responsivas por estado (Route tracker em primeiro lugar para o celular) — `docs/spec/screens.md`            |
| RNF03  | Agregação em SQL com índices e teste de 12 meses cronometrado — `docs/spec/business-rules.md`                     |
| RNF04  | Matriz de papéis por operação, aplicada na camada de serviço — `docs/spec/operations.md`                         |
| RNF05  | `audit_log` escrito na mesma transação de cada alteração de horários/composição/parâmetros                       |
| RNF06  | Minimização de dados, visibilidade por papel e política de remoção — `docs/spec/data-model.md`                   |

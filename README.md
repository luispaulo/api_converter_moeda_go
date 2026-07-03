#  API de Conversão de Moedas (BRL  Moedas Estrangeiras)

Uma API REST de alto desempenho desenvolvida em **Go** utilizando o framework **Gin Gonic**, projetada para converter valores em Real (BRL) para as 9 moedas mais utilizadas no mundo. A aplicação conta com cache automático no **PostgreSQL** para evitar limites de requisições externas e persistência física de histórico (logs) de todas as conversões.

---

##  Tecnologias Utilizadas

- **Linguagem**: Go (v1.26+)
- **Framework Web**: Gin Gonic
- **Banco de Dados**: PostgreSQL 15 (para cache e histórico de logs)
- **Driver de Banco**: pgx/v5 (com suporte a `pgxpool` para concorrência)
- **Live Reload (Hot Reload)**: Air (recompilação automática durante o desenvolvimento)
- **Orquestração de Containers**: Docker & Docker Compose
- **API de Câmbio Externa**: AwesomeAPI (sem necessidade de chaves/keys, taxas em tempo real)

---

## 📦 Como Subir o Projeto

### Pré-requisitos
- [Docker](https://www.docker.com/products/docker-desktop/) instalado e rodando.
- [Docker Compose](https://docs.docker.com/compose/) instalado.

### Passo a Passo

1. **Clonar/Entrar na pasta do projeto** e certificar-se de que os arquivos `.env` e `.air.toml` existem no diretório raiz.
2. **Iniciar os containers**:
   ```bash
   docker compose up --build
   ```
3. **Verificar os serviços ativos**:
   O Docker compose irá baixar, configurar e iniciar os seguintes containers:
   - **`docker-go-app`**: API Go rodando localmente na porta **`8080`** com recarregamento em tempo real ativo.
   - **`docker-go-db`**: Banco PostgreSQL rodando localmente exposto na porta **`5432`**.
   - **`docker-go-pgadmin`**: Interface administrativa do banco de dados na porta **`8082`**.

---

## 🔌 Endpoints da API

Uma vez que os contêineres estejam de pé, você pode utilizar os endpoints abaixo:

### 1. Conversão de Valores (`GET /api/convert`)
Converte um valor em Real (BRL) para moedas estrangeiras suportadas:
- **`USD`** (Dólar Americano)
- **`EUR`** (Euro)
- **`CAD`** (Dólar Canadense)
- **`GBP`** (Libra Esterlina)
- **`JPY`** (Iene Japonês)
- **`AUD`** (Dólar Australiano)
- **`CHF`** (Franco Suíço)
- **`CNY`** (Yuan Chinês)
- **`ARS`** (Peso Argentino)

#### Parâmetros de Query:
- `amount` *(Obrigatório)*: O valor decimal em BRL a ser convertido (ex: `amount=150.50`).
- `to` *(Opcional)*: A sigla da moeda de destino. **Se omitido, converte para todas as 9 moedas simultaneamente.**

#### Exemplo 1: Conversão em Lote (Todas as Moedas)
```bash
curl -s "http://localhost:8080/api/convert?amount=100"
```
**Resposta esperada:**
```json
{
  "original_amount": 100,
  "original_currency": "BRL",
  "conversions": [
    { "currency": "USD", "name": "Dólar Americano", "rate": 5.1849, "converted_amount": 19.286775058342493 },
    { "currency": "EUR", "name": "Euro", "rate": 5.92417, "converted_amount": 16.88000175552018 },
    { "currency": "CAD", "name": "Dólar Canadense", "rate": 3.64699, "converted_amount": 27.41987227823493 },
    ...
  ]
}
```

#### Exemplo 2: Conversão de Moeda Específica (BRL para USD)
```bash
curl -s "http://localhost:8080/api/convert?amount=100&to=USD"
```
**Resposta esperada:**
```json
{
  "converted_amount": 19.286775058342493,
  "name": "Dólar Americano",
  "original_amount": 100,
  "original_currency": "BRL",
  "rate": 5.1849,
  "target_currency": "USD"
}
```

---

### 2. Histórico de Conversões (`GET /api/logs`)
Retorna uma lista paginada de todas as conversões de moedas salvas no banco de dados.
#### Parâmetros de Query:
- `page` *(Opcional, default: 1)*: O número da página.
- `limit` *(Opcional, default: 10, max: 100)*: Quantidade de registros por página.

```bash
curl -s "http://localhost:8080/api/logs?page=1&limit=2"
```
**Resposta esperada:**
```json
{
  "conversions": [
    {
      "id": 10,
      "amount_brl": 100,
      "target_currency": "ARS",
      "rate": 0.003485,
      "converted_amount": 28694.4,
      "converted_at": "2026-07-03T22:06:09.998Z"
    },
    {
      "id": 9,
      "amount_brl": 100,
      "target_currency": "USD",
      "rate": 5.1849,
      "converted_amount": 19.29,
      "converted_at": "2026-07-03T22:06:09.998Z"
    }
  ],
  "pagination": {
    "limit": 2,
    "page": 1,
    "total": 10
  }
}
```

---

### 3. Listagem de Taxas e Cache (`GET /api/rates`)
Retorna as taxas de câmbio salvas no banco de dados e a última atualização.
```bash
curl -s http://localhost:8080/api/rates
```
> 💡 **Nota sobre Caching**: As taxas de câmbio ficam em cache no PostgreSQL por **30 minutos**. Consultas feitas nesse intervalo utilizam as taxas do banco local. Se o cache expirar ou estiver vazio, a API busca novas taxas na AwesomeAPI e atualiza o banco de dados de forma transparente.

---

### 4. Forçar Atualização de Câmbio (`POST /api/rates/update`)
Força a sincronização imediata com a AwesomeAPI e renova o cache do banco de dados na hora.
```bash
curl -s -X POST http://localhost:8080/api/rates/update
```

---

### 5. Health Check (`GET /health`)
Valida o status de saúde da aplicação e a conectividade física com o PostgreSQL.
```bash
curl -s http://localhost:8080/health
```

---

### 6. Rota de Ping (`GET /ping`)
Teste de latência simples para checar se o servidor web HTTP está de pé.
```bash
curl -s http://localhost:8080/ping
```

---

## 📊 Acessando o Banco de Dados (pgAdmin)

Para inspecionar as tabelas `exchange_rates` e `conversions` via navegador web:

1. Acesse: **[http://localhost:8082](http://localhost:8082)**.
2. Login com as credenciais do `.env`:
   - **E-mail**: `admin@admin.com`
   - **Senha**: `admin123`
3. Para registrar a conexão do banco no pgAdmin:
   - Clique com o botão direito em **"Servers"** ➡️ **"Register"** ➡️ **"Server..."**.
   - Na aba **General**, defina um nome (ex: `DB Local`).
   - Na aba **Connection**, insira os dados:
     - **Host name/address**: `db`
     - **Port**: `5432`
     - **Maintenance database**: `docker_go_db`
     - **Username**: `postgres`
     - **Password**: `postgres`
   - Clique em **Save**.

---

## ⚡ Desenvolvimento & Recompilação Dinâmica (Air)

O desenvolvimento dentro do container Docker utiliza a ferramenta **Air**. Sempre que qualquer arquivo `.go` do projeto for salvo, o container detectará a alteração e fará a recompilação automática:
```text
docker-go-app  |  • rebuilding...
docker-go-app  |  • running...
```
Isso dispensa a necessidade de reiniciar o Docker Compose manualmente ao alterar o código fonte.

# Go + Gin + PostgreSQL + pgAdmin no Docker 

Este projeto configura um ambiente de desenvolvimento completo em contêineres Docker para construir APIs utilizando Go (com o framework **Gin**), banco de dados **PostgreSQL** e **pgAdmin 4** como interface administrativa para o banco de dados.

O ambiente possui suporte a **Live Reloading (Hot Reload)** via [Air], o que significa que o código dentro do contêiner Docker será recompilado e atualizado automaticamente sempre que você salvar alterações em arquivos `.go`.

---

##  Pré-requisitos

Para executar este projeto, você precisará de:
*   [Docker](https://www.docker.com/products/docker-desktop/) instalado e rodando em sua máquina.
*   [Docker Compose](https://docs.docker.com/compose/) (normalmente incluído no Docker Desktop).

---

##  Estrutura de Arquivos do Projeto

*   [`main.go`] Código-fonte da aplicação que inicia o Gin e conecta ao Postgres.
*   [`Dockerfile`] Definição de imagem multi-estágio (Desenvolvimento com Air e Produção mínima).
*   [`docker-compose.yml`] Orquestração dos serviços (App, Banco de Dados, pgAdmin).
*   [`.air.toml`] Configurações do Air para monitoramento e recompilação em tempo real.
*   [`.env`] Variáveis de ambiente de configuração do banco e dos serviços.

---

##  Como Iniciar o Projeto

Siga os passos abaixo no seu terminal para levantar os contêineres:

1.  **Subir os serviços**:
    ```bash
    docker compose up --build
    ```

2.  **Verificar o status**:
    O Docker compose irá criar e iniciar 3 contêineres:
    *   **`docker-go-db`**: Banco Postgres rodando localmente na porta `5432`.
    *   **`docker-go-app`**: API Go rodando na porta `8080` com Hot Reload ativo.
    *   **`docker-go-pgadmin`**: Interface do pgAdmin rodando na porta `8082`.

---

##  Endpoints da API para Teste

Uma vez iniciado, você pode testar a API Go através das portas mapeadas na máquina local:

### 1. Endpoint `/ping`
Retorna uma resposta JSON simples para validar se o servidor HTTP está respondendo.
```bash
curl http://localhost:8080/ping
```
**Resposta esperada (JSON):**
```json
{
  "message": "pong"
}
```

### 2. Endpoint `/health` (Health Check)
Verifica se a aplicação Go consegue se conectar e fazer ping no banco de dados PostgreSQL.
```bash
curl http://localhost:8080/health
```
**Resposta esperada (JSON - Banco Funcionando):**
```json
{
  "database": "CONNECTED",
  "status": "UP"
}
```

---

##  Acessando e Configurando o pgAdmin

Para visualizar o banco de dados graficamente no navegador:

1.  Abra seu navegador no endereço: **[http://localhost:8082](http://localhost:8082)**.
2.  Faça login com as credenciais padrões definidas no arquivo [`.env`]:
    *   **E-mail**: `admin@admin.com`
    *   **Senha**: `admin123`
3.  **Adicionar o Servidor PostgreSQL no pgAdmin**:
    *   Clique com o botão direito em **"Servers"** ➡️ **"Register"** ➡️ **"Server..."**.
    *   Na aba **General**, defina um nome amigável (ex: `Docker Go DB`).
    *   Na aba **Connection**, preencha os dados de acordo com o `.env`:
        *   **Host name/address**: `db` *(Esse é o nome do serviço definido no docker-compose.yml)*
        *   **Port**: `5432`
        *   **Maintenance database**: `docker_go_db`
        *   **Username**: `postgres`
        *   **Password**: `postgres`
    *   Clique em **Save**. Agora você tem acesso completo ao banco pelo pgAdmin!

---

##  Como funciona o Live Reload (Air)

Se você editar qualquer arquivo `.go` do projeto (como o [`main.go`]) e salvá-lo, você verá no terminal uma mensagem parecida com:

```text
docker-go-app  |  • rebuilding...
docker-go-app  |  • running...
```

A aplicação reinicia de forma imediata dentro do contêiner, permitindo que você continue codificando sem ter que parar o docker-compose ou digitar comandos de compilação manuais.

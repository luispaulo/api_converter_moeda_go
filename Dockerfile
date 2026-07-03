# ==========================================
# 1. ESTÁGIO DE DESENVOLVIMENTO (HOT RELOAD)
# ==========================================
FROM golang:1.26-alpine AS dev

WORKDIR /app

# Instala dependências básicas de build (caso precise de CGO ou libs do SO)
RUN apk add --no-cache git build-base

# Instala o Air (gerenciador de live-reload mais atualizado)
RUN go install github.com/air-verse/air@latest

# Copia arquivos do Go Modules
COPY go.mod go.sum ./
RUN go mod download

# Copia o restante do código
COPY . .

# Porta padrão exposta
EXPOSE 8080

# Inicia o Air para assistir mudanças e rodar a aplicação
CMD ["air", "-c", ".air.toml"]

# ==========================================
# 2. ESTÁGIO DE CONSTRUÇÃO (BUILD DE PROD)
# ==========================================
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compila o binário estático otimizado para produção
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/api .

# ==========================================
# 3. ESTÁGIO DE PRODUÇÃO (IMAGEM MÍNIMA)
# ==========================================
FROM alpine:latest AS prod

WORKDIR /root/

# Copia o executável do estágio de build
COPY --from=builder /app/api .

# Expõe a porta de produção
EXPOSE 8080

# Comando para rodar a aplicação final
CMD ["./api"]

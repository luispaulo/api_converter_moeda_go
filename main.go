package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

var db *pgx.Conn

func main() {
	// Carrega o arquivo .env se ele existir (útil para desenvolvimento local fora do Docker)
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env não encontrado. Usando variáveis de ambiente do sistema.")
	}

	// Recupera configurações das variáveis de ambiente
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// Monta a string de conexão (DSN)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)

	// Tenta conectar ao banco de dados com retentativas (retry logic)
	// Isso evita que a aplicação falhe ao iniciar antes que o Postgres esteja pronto
	var err error
	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		log.Printf("Tentando conectar ao banco de dados (tentativa %d/%d)...", i, maxRetries)
		db, err = pgx.Connect(context.Background(), dsn)
		if err == nil {
			log.Println("Conectado ao banco de dados com sucesso!")
			break
		}
		log.Printf("Erro ao conectar ao banco de dados: %v. Aguardando 3 segundos...", err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("Falha crítica ao conectar ao banco de dados após %d tentativas: %v", maxRetries, err)
	}
	defer db.Close(context.Background())

	// Inicializa o roteador do Gin
	r := gin.Default()

	// Rota básica de ping
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Rota de saúde (health check) para validar a aplicação e conexão com o DB
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := db.Ping(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "DOWN",
				"details": fmt.Sprintf("Erro de conexão com o banco: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "UP",
			"database": "CONNECTED",
		})
	})

	// Inicia o servidor HTTP
	log.Printf("Servidor rodando na porta %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}

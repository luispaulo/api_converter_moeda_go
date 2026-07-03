package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"docker-go/database"
	"docker-go/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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

	// Inicializa conexão pool com o Banco de Dados e migrações
	if err := database.InitDB(dsn); err != nil {
		log.Fatalf("Erro ao inicializar banco de dados: %v", err)
	}
	defer database.CloseDB()

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

		err := database.DB.Ping(ctx)
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

	// Grupo de rotas da API
	api := r.Group("/api")
	{
		api.GET("/rates", handlers.GetRates)
		api.GET("/convert", handlers.Convert)
		api.GET("/logs", handlers.GetLogs)
		api.POST("/rates/update", handlers.UpdateRates)
	}

	// Inicia o servidor HTTP
	log.Printf("Servidor rodando na porta %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}


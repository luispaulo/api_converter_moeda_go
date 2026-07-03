package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB é o pool de conexões global com o banco de dados
var DB *pgxpool.Pool

// InitDB inicializa o pool de conexões do pgxpool e roda as migrations iniciais
func InitDB(dsn string) error {
	var err error
	maxRetries := 10

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("erro ao analisar string de conexão: %w", err)
	}

	// Tenta conectar ao banco de dados com retentativas (retry logic)
	for i := 1; i <= maxRetries; i++ {
		log.Printf("Tentando conectar ao banco de dados (tentativa %d/%d)...", i, maxRetries)
		DB, err = pgxpool.NewWithConfig(context.Background(), config)
		if err == nil {
			// Executa um Ping simples para garantir que a conexão foi realmente estabelecida
			err = DB.Ping(context.Background())
			if err == nil {
				log.Println("Conectado ao banco de dados PostgreSQL com sucesso!")
				break
			}
			DB.Close()
		}
		log.Printf("Erro ao conectar ao banco de dados: %v. Aguardando 3 segundos...", err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("falha crítica ao conectar ao banco de dados após %d tentativas: %w", maxRetries, err)
	}

	// Executa as migrations para criar as tabelas necessárias
	if err := runMigrations(); err != nil {
		return fmt.Errorf("erro ao executar migrations: %w", err)
	}

	return nil
}

// CloseDB fecha o pool de conexões com segurança
func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("Pool de conexões com o banco de dados encerrado.")
	}
}

// runMigrations executa as instruções DDL iniciais para criar as tabelas
func runMigrations() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tabela para cachear as taxas de câmbio
	createRatesTable := `
	CREATE TABLE IF NOT EXISTS exchange_rates (
		code VARCHAR(10) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		rate NUMERIC(15, 6) NOT NULL,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	// Tabela para log/histórico de conversões executadas
	createConversionsTable := `
	CREATE TABLE IF NOT EXISTS conversions (
		id SERIAL PRIMARY KEY,
		amount_brl NUMERIC(15, 2) NOT NULL,
		target_currency VARCHAR(10) NOT NULL,
		rate NUMERIC(15, 6) NOT NULL,
		converted_amount NUMERIC(15, 2) NOT NULL,
		converted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	log.Println("Executando migrations no banco de dados...")

	_, err := DB.Exec(ctx, createRatesTable)
	if err != nil {
		return fmt.Errorf("falha ao criar tabela exchange_rates: %w", err)
	}

	_, err = DB.Exec(ctx, createConversionsTable)
	if err != nil {
		return fmt.Errorf("falha ao criar tabela conversions: %w", err)
	}

	log.Println("Migrations executadas com sucesso!")
	return nil
}

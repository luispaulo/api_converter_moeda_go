package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"docker-go/database"
	"docker-go/models"
)

const (
	awesomeAPIURL = "https://economia.awesomeapi.com.br/json/last/USD-BRL,EUR-BRL,CAD-BRL,GBP-BRL,JPY-BRL,AUD-BRL,CHF-BRL,CNY-BRL,ARS-BRL"
	cacheDuration = 30 * time.Minute
)

// SupportedCurrencies lista as moedas que suportamos para conversão
var SupportedCurrencies = []string{"USD", "EUR", "CAD", "GBP", "JPY", "AUD", "CHF", "CNY", "ARS"}

// FetchExternalRates busca as taxas mais recentes da AwesomeAPI e as salva no banco de dados
func FetchExternalRates() ([]models.ExchangeRate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Buscando taxas externas na AwesomeAPI...")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, awesomeAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição externa: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar AwesomeAPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resposta inesperada da AwesomeAPI: status %d", resp.StatusCode)
	}

	var rawRates map[string]models.AwesomeAPICurrencyInfo
	if err := json.NewDecoder(resp.Body).Decode(&rawRates); err != nil {
		return nil, fmt.Errorf("erro ao decodificar JSON da AwesomeAPI: %w", err)
	}

	var rates []models.ExchangeRate
	now := time.Now()

	// Inicia transação para atualizar todas as taxas
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, info := range rawRates {
		// Converte o valor de 'bid' (string) para float64
		rateValue, err := strconv.ParseFloat(info.Bid, 64)
		if err != nil {
			log.Printf("Aviso: erro ao converter taxa '%s' para float64: %v. Ignorando.", info.Bid, err)
			continue
		}

		// Limpa o nome da moeda (AwesomeAPI retorna ex: "Dólar Americano/Real Brasileiro")
		cleanName := strings.Split(info.Name, "/")[0]

		rate := models.ExchangeRate{
			Code:      info.Code,
			Name:      cleanName,
			Rate:      rateValue,
			UpdatedAt: now,
		}

		// Upsert no banco de dados
		query := `
		INSERT INTO exchange_rates (code, name, rate, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (code) 
		DO UPDATE SET name = EXCLUDED.name, rate = EXCLUDED.rate, updated_at = EXCLUDED.updated_at;`

		_, err = tx.Exec(ctx, query, rate.Code, rate.Name, rate.Rate, rate.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("erro ao salvar taxa %s no banco: %w", rate.Code, err)
		}

		rates = append(rates, rate)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("erro ao comitar transação de taxas: %w", err)
	}

	log.Printf("Taxas de %d moedas atualizadas no banco de dados.", len(rates))
	return rates, nil
}

// GetOrUpdateRates busca as taxas do banco de dados.
// Se as taxas estiverem vazias ou expiradas (> 30 minutos), busca novas na AwesomeAPI.
func GetOrUpdateRates() ([]models.ExchangeRate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rates []models.ExchangeRate
	query := `SELECT code, name, rate, updated_at FROM exchange_rates;`

	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar taxas no banco: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rate models.ExchangeRate
		if err := rows.Scan(&rate.Code, &rate.Name, &rate.Rate, &rate.UpdatedAt); err != nil {
			return nil, fmt.Errorf("erro ao ler linha de taxa: %w", err)
		}
		rates = append(rates, rate)
	}

	// Se não houver taxas no banco, ou se a mais antiga tiver mais de 30 minutos, atualizamos
	needUpdate := len(rates) == 0
	if !needUpdate {
		// Verifica se alguma taxa expirou
		for _, r := range rates {
			if time.Since(r.UpdatedAt) > cacheDuration {
				needUpdate = true
				break
			}
		}
	}

	if needUpdate {
		log.Println("Cache expirado ou inexistente. Atualizando taxas de câmbio...")
		updatedRates, err := FetchExternalRates()
		if err != nil {
			// Se falhar a atualização externa mas tivermos taxas antigas no banco, usamos as antigas como fallback
			if len(rates) > 0 {
				log.Printf("Erro ao buscar taxas externas: %v. Usando taxas cacheadas expiradas como fallback.", err)
				return rates, nil
			}
			return nil, err
		}
		return updatedRates, nil
	}

	return rates, nil
}

// ConvertBRL realiza a conversão de um valor em Real (BRL) para uma moeda específica ou para todas
func ConvertBRL(amountBRL float64, targetCurrency string) ([]models.ConversionResult, error) {
	rates, err := GetOrUpdateRates()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter taxas de câmbio: %w", err)
	}

	// Cria um mapa para facilitar a busca rápida das taxas por código de moeda
	ratesMap := make(map[string]models.ExchangeRate)
	for _, r := range rates {
		ratesMap[r.Code] = r
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()

	// Se a moeda destino for específica
	if targetCurrency != "" {
		targetCurrency = strings.ToUpper(targetCurrency)
		rate, ok := ratesMap[targetCurrency]
		if !ok {
			return nil, errors.New("moeda de destino não suportada ou não encontrada")
		}

		// A AwesomeAPI retorna o preço de 1 unidade da moeda estrangeira em BRL (Ex: 1 USD = 5.18 BRL)
		// Então: Valor convertido = Valor BRL / Taxa
		convertedAmount := amountBRL / rate.Rate

		// Grava o log no banco de dados
		logQuery := `
		INSERT INTO conversions (amount_brl, target_currency, rate, converted_amount, converted_at)
		VALUES ($1, $2, $3, $4, $5);`
		
		_, err := database.DB.Exec(ctx, logQuery, amountBRL, targetCurrency, rate.Rate, convertedAmount, now)
		if err != nil {
			log.Printf("Aviso: erro ao gravar log de conversão no banco: %v", err)
		}

		return []models.ConversionResult{
			{
				Currency:        targetCurrency,
				Name:            rate.Name,
				Rate:            rate.Rate,
				ConvertedAmount: convertedAmount,
			},
		}, nil
	}

	// Se for converter para todas as moedas
	var results []models.ConversionResult

	// Abre transação para salvar múltiplos logs de conversão de forma atômica
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		log.Printf("Aviso: erro ao iniciar transação para logs: %v", err)
	}
	defer tx.Rollback(ctx)

	for _, code := range SupportedCurrencies {
		rate, ok := ratesMap[code]
		if !ok {
			continue
		}

		convertedAmount := amountBRL / rate.Rate
		results = append(results, models.ConversionResult{
			Currency:        code,
			Name:            rate.Name,
			Rate:            rate.Rate,
			ConvertedAmount: convertedAmount,
		})

		// Grava log
		if tx != nil {
			logQuery := `
			INSERT INTO conversions (amount_brl, target_currency, rate, converted_amount, converted_at)
			VALUES ($1, $2, $3, $4, $5);`
			_, err = tx.Exec(ctx, logQuery, amountBRL, code, rate.Rate, convertedAmount, now)
			if err != nil {
				log.Printf("Aviso: erro ao inserir log na transação para %s: %v", code, err)
			}
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			log.Printf("Aviso: erro ao comitar transação de logs de conversão: %v", err)
		}
	}

	return results, nil
}

// GetConversionLogs recupera o histórico de conversões realizadas de forma paginada
func GetConversionLogs(page, limit int) ([]models.ConversionLog, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Obtém total de registros
	var total int
	countQuery := "SELECT COUNT(*) FROM conversions;"
	err := database.DB.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao contar logs de conversão: %w", err)
	}

	// Obtém os registros paginados
	query := `
	SELECT id, amount_brl, target_currency, rate, converted_amount, converted_at
	FROM conversions
	ORDER BY converted_at DESC
	LIMIT $1 OFFSET $2;`

	rows, err := database.DB.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao buscar logs de conversão: %w", err)
	}
	defer rows.Close()

	var logs []models.ConversionLog
	for rows.Next() {
		var l models.ConversionLog
		err := rows.Scan(&l.ID, &l.AmountBRL, &l.TargetCurrency, &l.Rate, &l.ConvertedAmount, &l.ConvertedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao ler linha de log de conversão: %w", err)
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}

package handlers

import (
	"net/http"
	"strconv"

	"docker-go/services"

	"github.com/gin-gonic/gin"
)

// GetRates retorna todas as taxas de câmbio salvas no banco
func GetRates(c *gin.Context) {
	rates, err := services.GetOrUpdateRates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Falha ao recuperar taxas de câmbio",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rates": rates,
	})
}

// Convert realiza a conversão de BRL para uma ou todas as moedas
func Convert(c *gin.Context) {
	amountStr := c.Query("amount")
	if amountStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "O parâmetro 'amount' é obrigatório",
		})
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "O parâmetro 'amount' deve ser um valor numérico positivo válido",
		})
		return
	}

	to := c.Query("to")

	results, err := services.ConvertBRL(amount, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Falha ao realizar conversão",
			"details": err.Error(),
		})
		return
	}

	// Se a busca foi por moeda específica, retorna um objeto plano ou array
	if to != "" {
		c.JSON(http.StatusOK, gin.H{
			"original_amount":   amount,
			"original_currency": "BRL",
			"target_currency":   results[0].Currency,
			"name":              results[0].Name,
			"rate":              results[0].Rate,
			"converted_amount":  results[0].ConvertedAmount,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"original_amount":   amount,
		"original_currency": "BRL",
		"conversions":       results,
	})
}

// UpdateRates força a atualização das taxas no banco de dados com a AwesomeAPI
func UpdateRates(c *gin.Context) {
	rates, err := services.FetchExternalRates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Falha ao forçar atualização externa de taxas",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Taxas atualizadas com sucesso!",
		"count":   len(rates),
	})
}

// GetLogs retorna o histórico paginado das conversões realizadas
func GetLogs(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	logs, total, err := services.GetConversionLogs(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Falha ao buscar logs de conversão",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversions": logs,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

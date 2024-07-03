// server.go

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFile         = "cotacoes.db"
	timeoutAPI     = 200 * time.Millisecond
	timeoutDBWrite = 10 * time.Millisecond
)

type Cotacao struct {
	Bid string `json:"bid"`
}

func main() {
	db, err := sqlx.Connect("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	createTable := `
		CREATE TABLE IF NOT EXISTS cotacoes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bid REAL,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := db.Exec(createTable); err != nil {
		log.Fatalf("Erro ao criar tabela: %v", err)
	}

	http.HandleFunc("/cotacao", handleCotacaoRequest)

	addr := ":8080"
	log.Printf("Servidor iniciado. Escutando na porta %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleCotacaoRequest(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeoutAPI)
	defer cancel()

	cotacao, err := getCotacaoFromAPI(ctx)
	if err != nil {
		http.Error(w, "Erro ao obter cotação do dólar", http.StatusInternalServerError)
		return
	}

	ctxDB, cancelDB := context.WithTimeout(ctx, timeoutDBWrite)
	defer cancelDB()
	if err := saveCotacao(ctxDB, cotacao); err != nil {
		log.Printf("Erro ao salvar cotação no banco de dados: %v", err)
	}

	resultado := map[string]string{"bid": cotacao.Bid}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resultado); err != nil {
		http.Error(w, "Erro ao serializar resposta", http.StatusInternalServerError)
		return
	}
}

func getCotacaoFromAPI(ctx context.Context) (*Cotacao, error) {
	client := http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar requisição HTTP: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao realizar requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("a API retornou um status não esperado: %v", resp.Status)
	}

	body, error := io.ReadAll(resp.Body)
	if error != nil {
		return nil, fmt.Errorf("falha ao ler o response: %v", err)
	}

	var cotacoes map[string]Cotacao
	if err := json.Unmarshal(body, &cotacoes); err != nil {
		return nil, fmt.Errorf("falha ao decodificar resposta JSON: %v", err)
	}

	cotacao, ok := cotacoes["USDBRL"]
	if !ok {
		return nil, fmt.Errorf("campo 'USDBRL' não encontrado na resposta da API")
	}

	return &cotacao, nil
}

func saveCotacao(ctx context.Context, cotacao *Cotacao) error {
	db, err := sqlx.Connect("sqlite3", dbFile)
	if err != nil {
		return fmt.Errorf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	query := "INSERT INTO cotacoes (bid) VALUES (:bid)"
	_, err = db.NamedExecContext(ctx, query, cotacao)
	if err != nil {
		return fmt.Errorf("falha ao inserir cotação no banco de dados: %v", err)
	}
	return nil
}

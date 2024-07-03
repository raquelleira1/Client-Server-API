// client.go

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Bid struct {
	Bid string `json:"bid"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Erro: servidor retornou status %d", resp.StatusCode)
	}

	var cotacao Bid
	if err := json.NewDecoder(resp.Body).Decode(&cotacao); err != nil {
		log.Fatalf("Erro ao decodificar resposta JSON: %v", err)
	}

	if err := salvarCotacaoEmArquivo(cotacao); err != nil {
		log.Fatalf("Erro ao salvar cotação em arquivo: %v", err)
	}

	log.Printf("Cotação salva com sucesso em cotacao.txt: Dólar:" + cotacao.Bid)
}

func salvarCotacaoEmArquivo(cotacao Bid) error {
	arquivo := "cotacao.txt"
	arquivoOut, err := os.Create(arquivo)
	if err != nil {
		return fmt.Errorf("falha ao criar arquivo: %v", err)
	}
	defer arquivoOut.Close()

	if _, err := arquivoOut.WriteString("Dólar:" + cotacao.Bid); err != nil {
		return fmt.Errorf("falha ao escrever no arquivo: %v", err)
	}

	return nil
}

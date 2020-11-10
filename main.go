package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RogerioML/plp"
)

const (
	layoutMysql = "2006-01-02 15:04:05"
)

//Config contem as configurações globais do app
type Config struct {
	Dsn       string `json:"dsn"`
	Intervalo int    `json:"intervalo"`
	Wsdl      string `json:"wsdl"`
	MaxConn   int    `json:"conexoes"`
	Servico   string `json:"servico"`
	Cnpj      string `json:"cnpj"`
	Qtd       int    `json:"qtdEtiquetas"`
	User      string `json:"usuario"`
	Pass      string `json:"senha"`
}

func main() {
	var config Config

	arq, err := os.Open("config.json")
	if err != nil {
		panic(err.Error())
	}

	jsonParser := json.NewDecoder(arq)
	if err = jsonParser.Decode(&config); err != nil {
		log.Fatal(err.Error())
	}

	timer := time.NewTicker(time.Duration(config.Intervalo) * time.Second)
	for {
		select {
		case <-timer.C:
			//solicita uma faixa de etiquetas
			now := time.Now()

			cliente, err := plp.ConsultaClientePorContratoResponse(config.Wsdl, "912208555", "10")
			if err != nil {
				log.Printf("falha ao consultar cliente: %s: ", err.Error())
				continue
			}
			log.Printf("consultaCliente %.3f cliente obtido %s", time.Since(now).Seconds(), cliente.Body.ConsultaClientePorContratoResponse.Cliente.NomeFantasia)

			faixa, err := plp.SolicitaEtiquetas(config.Wsdl, config.Servico, config.Cnpj, config.Qtd, config.User, config.Pass)
			if err != nil {
				log.Printf("falha ao obter etiquetas: %s", err.Error())
				continue
			}
			log.Printf("solicitaEtiquetas %.3f faixa obtida %s", time.Since(now).Seconds(), faixa)
			if len(faixa) < 1 {
				log.Println("faixa de etiquetas em formato invalido")
				continue
			}
			//solicita o digito verificador para  faixa gerada
			now = time.Now()
			etq := faixa[:13]
			verificador, err := plp.GeraDigitoVerificadorEtiquetas(config.Wsdl, etq)
			if err != nil {
				log.Printf("falha ao obter digito verificador: %s", err.Error())
				continue
			}
			etqComVerificador := strings.Replace(etq, " ", strconv.Itoa(verificador), -1)
			log.Printf("geraDigitoVerificadorEtiquetas %.3f etiqueta completa: %s", time.Since(now).Seconds(), etqComVerificador)

			//gerar a plp com a etiqueta obtida
			etqSemVerificador := strings.Replace(etq, " ", "", -1)
			now = time.Now()
			plpNu, status, err := plp.FechaPlpVariosServicos(config.Wsdl, etqComVerificador, etqSemVerificador)
			if err != nil {
				fmt.Println(now.Format(layoutMysql) + " erro ao fechar PLP: " + status + " " + err.Error())
				continue
			}
			log.Printf("fechaPlpVariosServicos %.3f %s plp obtida: %s", time.Since(now).Seconds(), status, plpNu)
		}
	}
}

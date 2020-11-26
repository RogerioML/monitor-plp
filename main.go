package main

import (
	"encoding/json"
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
	Dsn          string `json:"dsn"`
	Intervalo    int    `json:"intervalo"`
	Wsdl         string `json:"wsdl"`
	MaxConn      int    `json:"conexoes"`
	Contrato     string `json:"contrato"`
	Cartao       string `json:"cartao"`
	Servico      string `json:"servico"`
	Cnpj         string `json:"cnpj"`
	Qtd          int    `json:"qtdEtiquetas"`
	User         string `json:"usuario"`
	Pass         string `json:"senha"`
	IDPlpCliente string `json:"idPlpCliente"`
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
	plp.Wsdl = config.Wsdl

	timer := time.NewTicker(time.Duration(config.Intervalo) * time.Second)
	for {
		select {
		case <-timer.C:
			//solicita uma faixa de etiquetas
			now := time.Now()

			endereco, err := plp.ConsultaCEP("71917360")
			if err != nil {
				log.Printf("falha ao consultar cep: %s", err.Error())
				continue
			}
			log.Printf("buscaCEP %.3f endereco obtido: %s %s %s - %s", time.Since(now).Seconds(),
				endereco.Body.ConsultaCEPResponse.Return.Endereco,
				endereco.Body.ConsultaCEPResponse.Return.Bairro,
				endereco.Body.ConsultaCEPResponse.Return.Cidade,
				endereco.Body.ConsultaCEPResponse.Return.UF,
			)

			servicos, err := plp.BuscaServicos("9912408500", "0072922621", config.User, config.Pass)
			if err != nil {
				log.Printf("falha ao consultar cliente: %s: ", err.Error())
				continue
			}
			log.Printf("buscaServicos %.3f total de %d servicos obtidos", time.Since(now).Seconds(), len(servicos.Body.BuscaServicosResponse.Return))

			faixa, err := plp.SolicitaEtiquetas(config.Servico, config.Cnpj, config.Qtd, config.User, config.Pass)
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
			verificador, err := plp.GeraDigitoVerificadorEtiquetas(etq)
			if err != nil {
				log.Printf("falha ao obter digito verificador: %s", err.Error())
				continue
			}
			etqComVerificador := strings.Replace(etq, " ", strconv.Itoa(verificador), -1)
			log.Printf("geraDigitoVerificadorEtiquetas %.3f etiqueta completa: %s", time.Since(now).Seconds(), etqComVerificador)

			//gerar a plp com a etiqueta obtida
			etqSemVerificador := strings.Replace(etq, " ", "", -1)
			now = time.Now()
			plpNu, err := plp.FechaPlpVariosServicos(etqComVerificador, etqSemVerificador, config.IDPlpCliente, config.Cartao, config.User, config.Pass)
			if err != nil {
				log.Println("erro ao fechar PLP: " + err.Error())
				continue
			}
			log.Printf("fechaPlpVariosServicos %.3f plp obtida: %s", time.Since(now).Seconds(), plpNu)

			_, err = plp.SolicitaPLP(plpNu, etqComVerificador, config.User, config.Pass)
			if err != nil {
				log.Println("erro ao obter PLP: " + err.Error())
				continue
			}
			log.Printf("solicitaPLP %.3f plp obtida com sucesso", time.Since(now).Seconds())
		}
	}
}

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
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

var fecharPlp = flag.Bool("plp", true, "definir se deve ou não fechar plp")
var qtdTestes = flag.Int("q", 1, "quantidade de testes a realizar")
var xmlName = flag.String("a", "plp.xml", "nome do arquivo xml a ser considerado")
var configFile = flag.String("c", "config.json", "nome do arquivo json")

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

func testaAmbiente(config Config, xml string) {
	for i := 0; i < *qtdTestes; i++ {
		go func() {
			//solicita uma faixa de etiquetas
			now := time.Now()

			cliente, err := plp.BuscaCliente(config.Contrato, config.Cartao, config.User, config.Pass)
			if err != nil {
				log.Printf("falha ao consultar cliente: %s", err.Error())
				return
			}
			log.Printf("buscaCliente %.3f cliente obtido: %s\n", time.Since(now).Seconds(), cliente.Body.BuscaClienteResponse.Return.Cnpj)
			config.Cnpj = cliente.Body.BuscaClienteResponse.Return.Cnpj

			servicos, err := plp.BuscaServicos(config.Contrato, config.Cartao, config.User, config.Pass)
			if err != nil {
				log.Printf("falha ao buscar servicos: %s: ", err.Error())
				return
			}
			log.Printf("buscaServicos %.3f total de %d servicos obtidos: \n", time.Since(now).Seconds(), len(servicos.Body.BuscaServicosResponse.Return))

			endereco, err := plp.ConsultaCEP("71917360")
			if err != nil {
				log.Printf("falha ao consultar cep: %s", err.Error())
				return
			}

			/*
				jsonServicos, err := json.MarshalIndent(servicos, "", "	")
				if err != nil {
					log.Printf("falha ao converter servicos para JSON: %s", err.Error())
					return
				}
				log.Println(string(jsonServicos))*/

			log.Printf("buscaCEP %.3f endereco obtido: %s %s %s - %s", time.Since(now).Seconds(),
				endereco.Body.ConsultaCEPResponse.Return.Endereco,
				endereco.Body.ConsultaCEPResponse.Return.Bairro,
				endereco.Body.ConsultaCEPResponse.Return.Cidade,
				endereco.Body.ConsultaCEPResponse.Return.UF,
			)

			faixa, err := plp.SolicitaEtiquetas(config.Servico, config.Cnpj, config.Qtd, config.User, config.Pass)
			if err != nil {
				log.Printf("falha ao obter etiquetas: %s", err.Error())
				return
			}
			log.Printf("solicitaEtiquetas %.3f faixa obtida %s", time.Since(now).Seconds(), faixa)
			if len(faixa) < 1 {
				log.Println("faixa de etiquetas em formato invalido")
				return
			}
			//solicita o digito verificador para  faixa gerada
			now = time.Now()
			etq := faixa[:13]
			verificador, err := plp.GeraDigitoVerificadorEtiquetas(etq)
			if err != nil {
				log.Printf("erro: main 3: falha ao obter digito verificador: %s", err.Error())
				return
			}
			etqComVerificador := strings.Replace(etq, " ", strconv.Itoa(verificador), -1)
			log.Printf("geraDigitoVerificadorEtiquetas %.3f etiqueta completa: %s", time.Since(now).Seconds(), etqComVerificador)

			//gerar a plp com a etiqueta obtida
			etqSemVerificador := strings.Replace(etq, " ", "", -1)
			if *fecharPlp {
				now = time.Now()
				plpNu, err := plp.FechaPlpVariosServicos(xml, etqComVerificador, etqSemVerificador, config.IDPlpCliente, config.Cartao, config.User, config.Pass)
				if err != nil {
					log.Println("erro ao fechar PLP: " + err.Error())
					return
				}
				log.Printf("fechaPlpVariosServicos %.3f plp obtida: %s", time.Since(now).Seconds(), plpNu)

				_, err = plp.SolicitaPLP(plpNu, etqComVerificador, config.User, config.Pass)
				if err != nil {
					log.Println("erro ao obter PLP: " + err.Error())
					return
				}
				log.Printf("solicitaPLP %.3f plp obtida com sucesso", time.Since(now).Seconds())
				now = time.Now()
				err = plp.CancelarObjeto(etqComVerificador, plpNu, config.User, config.Pass)
				if err != nil {
					log.Println("erro ao cancelar objeto: " + err.Error())
					return
				}
				log.Printf("cancelarObjeto %.3f etiqueta %s cancelada com sucesso\n", time.Since(now).Seconds(), etqComVerificador)
			}
		}()
	}
}

func main() {
	flag.Parse()
	var config Config

	arq, err := os.Open(*configFile)
	if err != nil {
		panic(err.Error())
	}

	xmlFile, err := ioutil.ReadFile(*xmlName)
	if err != nil {
		panic(err.Error())
	}
	xml := string(xmlFile)

	jsonParser := json.NewDecoder(arq)
	if err = jsonParser.Decode(&config); err != nil {
		log.Fatal(err.Error())
	}
	plp.Wsdl = config.Wsdl
	plp.User = config.User
	plp.Pass = config.Pass

	timer := time.NewTicker(time.Duration(config.Intervalo) * time.Second)
	for {
		select {
		case <-timer.C:
			log.Println("<===========================iniciando ciclo de teste==========================>")
			testaAmbiente(config, xml)
		}
	}
}

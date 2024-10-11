#Aplicação de Consulta de Clima por CEP

Este projeto consiste em dois microserviços em Go, Service A e Service B, que permitem obter informações climáticas de uma cidade a partir de um CEP brasileiro. A aplicação utiliza OpenTelemetry para instrumentação e Zipkin para tracing distribuído.

- Sumário
- Descrição da Aplicação
- Arquitetura
- Pré-requisitos
- Configuração
- Executando a Aplicação
- Testando a Aplicação
- Monitorando com Zipkin
- Estrutura dos Serviços
- Observações Importantes
- Licença
- Descrição da Aplicação

## A aplicação permite que o usuário forneça um CEP (Código de Endereçamento Postal) brasileiro e receba como resposta:

## Nome da cidade correspondente ao CEP.
Temperatura atual da cidade em Celsius, Fahrenheit e Kelvin.

## Arquitetura

### Service A: Recebe o CEP do usuário, valida o formato e encaminha a requisição para o Service B.
### Service B: Recebe o CEP, consulta a API ViaCEP para obter o nome da cidade e, em seguida, consulta a WeatherAPI para obter as informações climáticas da cidade.
- Ambos os serviços estão instrumentados com OpenTelemetry para tracing distribuído, e utilizam o Zipkin como coletor e visualizador de traces.

## Pré-requisitos
Docker e Docker Compose instalados na máquina.
1. Clonar o Repositório
bash
Copiar código
git clone https://github.com/seu-usuario/seu-repositorio.git
cd seu-repositorio

## Executando a Aplicação
Na raiz do projeto, execute o seguinte comando para construir e iniciar os serviços:

bash '''
Copiar código
docker-compose up --build
''' 
Isso irá:

- Construir as imagens Docker dos serviços A e B.
- Iniciar os serviços A, B e o Zipkin.
- Mapear as portas necessárias:
- Service A: localhost:8080
- Zipkin: localhost:9411

## Testando a Aplicação
Você pode testar a aplicação utilizando o curl ou ferramentas como Postman e Insomnia.

### 1. Requisição com CEP Válido

curl -X POST \
  http://localhost:8080/cep \
  -H 'Content-Type: application/json' \
  -d '{ "cep": "01001000" }'
Resposta Esperada:

json
{
  "city": "São Paulo",
  "tempC": 19.9,
  "tempF": 67.9,
  "tempK": 293.05
}

## Notas:

Os valores de temperatura podem variar conforme as condições climáticas atuais.
O CEP 01001000 é um CEP válido para São Paulo.

### 2. Requisição com CEP com Hífen
A aplicação aceita CEPs com ou sem hífen.

curl -X POST \
  http://localhost:8080/cep \
  -H 'Content-Type: application/json' \
  -d '{ "cep": "01001-000" }'

### 3. Requisição com CEP Inválido

curl -X POST \
  http://localhost:8080/cep \
  -H 'Content-Type: application/json' \
  -d '{ "cep": "123" }'
Resposta Esperada:

invalid zipcode

## Monitorando com Zipkin
Acesse o Zipkin para visualizar os traces das requisições:

URL: http://localhost:9411
No Zipkin, você pode:

- Verificar as requisições realizadas pelos serviços.
- Visualizar a distribuição de tempo entre os serviços A e B.
- Identificar possíveis gargalos ou erros nas requisições.
- Estrutura dos Serviços
- Service A
- Linguagem: Go
- Porta: 8080

### Responsabilidades:
Receber requisições POST com um CEP no formato JSON.
Validar o formato do CEP (aceita com ou sem hífen).
Encaminhar a requisição para o Service B.
Retornar a resposta final ao cliente.
Service B
Linguagem: Go
Porta: 8081
Responsabilidades:
Receber o CEP do Service A.
Consultar a API ViaCEP para obter o nome da cidade.
Consultar a WeatherAPI para obter as informações climáticas.
Retornar os dados para o Service A.
OpenTelemetry e Zipkin
Instrumentação: Os serviços utilizam OpenTelemetry para coletar dados de tracing.
Coletor: O Zipkin recebe e exibe os dados coletados pelos serviços.
Benefícios:
Monitoramento distribuído.
Análise de performance e identificação de problemas.
Observações Importantes
Variáveis de Ambiente:
Certifique-se de que a variável WEATHER_API_KEY está definida corretamente no docker-compose.yml.
Dependências:
Os serviços utilizam módulos Go e dependem das APIs ViaCEP e WeatherAPI.
Limitações da API:
A WeatherAPI possui limites de requisições para contas gratuitas. Verifique seu plano para evitar exceder o limite.

### Licença
Este projeto está licenciado sob a MIT License.

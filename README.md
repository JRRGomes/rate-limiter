# Rate Limiter

A Go-based rate limiting service that provides different rate limits based on token types (public, premium, admin) and IP addresses. The service uses Redis for state management and provides a simple HTTP middleware.

## Desafio

### Objetivo
Desenvolver um rate limiter em Go que possa ser configurado para limitar o número máximo de requisições por segundo com base em um endereço IP específico ou em um token de acesso.

### Descrição
O objetivo deste desafio é criar um rate limiter em Go que possa ser utilizado para controlar o tráfego de requisições para um serviço web. O rate limiter deve ser capaz de limitar o número de requisições com base em dois critérios:

1. **Endereço IP**: O rate limiter deve restringir o número de requisições recebidas de um único endereço IP dentro de um intervalo de tempo definido.
2. **Token de Acesso**: O rate limiter deve também limitar as requisições baseadas em um token de acesso único, permitindo diferentes limites de tempo de expiração para diferentes tokens. O Token deve ser informado no header no seguinte formato:
   ```plaintext
   API_KEY: <TOKEN>
   ```
3. **Prioridade do Token sobre o IP**: Se um limite específico de token estiver configurado, ele deve se sobrepor ao limite por IP. Exemplo: Se o limite por IP é de 10 req/s e o de um determinado token é de 100 req/s, o rate limiter deve utilizar as informações do token.

### Requisitos
- O rate limiter deve funcionar como um **middleware** injetado ao servidor web.
- Deve permitir a **configuração do número máximo de requisições** permitidas por segundo.
- Deve ter a opção de **configurar o tempo de bloqueio** do IP ou do Token caso a quantidade de requisições tenha sido excedida.
- As configurações de limite devem ser realizadas via **variáveis de ambiente** ou em um arquivo `.env` na pasta raiz.
- Deve ser possível configurar o rate limiter tanto para **limitação por IP** quanto por **token de acesso**.
- O sistema deve responder adequadamente quando o limite é excedido:
  - **Código HTTP**: `429`
  - **Mensagem**: `you have reached the maximum number of requests or actions allowed within a certain time frame`
- Todas as informações do rate limiter devem ser **armazenadas e consultadas de um banco de dados Redis**.
- Utilize `docker-compose` para subir o Redis.
- Implementar uma **strategy** que permita trocar facilmente o Redis por outro mecanismo de persistência.
- A lógica do limiter deve estar **separada do middleware**.

### Exemplos
1. **Limitação por IP**
   - Suponha que o rate limiter esteja configurado para permitir no máximo `5` requisições por segundo por IP.
   - Se o IP `192.168.1.1` enviar `6` requisições em um segundo, a sexta requisição deve ser bloqueada.

2. **Limitação por Token**
   - Se um token `abc123` tiver um limite configurado de `10` requisições por segundo e enviar `11` requisições nesse intervalo, a décima primeira deve ser bloqueada.

3. **Tempo de Expiração**
   - Nos dois casos acima, as próximas requisições poderão ser realizadas somente quando o tempo total de expiração ocorrer.
   - Exemplo: Se o tempo de expiração for de **5 minutos**, o IP ou Token poderá realizar novas requisições somente após esse tempo.

### Dicas
- Teste seu rate limiter sob diferentes condições de carga para garantir que ele funcione conforme esperado em situações de alto tráfego.

### Entrega
- O **código-fonte** completo da implementação.
- **Documentação** explicando como o rate limiter funciona e como ele pode ser configurado.
- **Testes automatizados** demonstrando a eficácia e a robustez do rate limiter.
- Utilize **Docker/Docker-Compose** para facilitar os testes da aplicação.
- O servidor web deve responder na **porta 8080**.

### Rate Limits

- IP-based: 20 requests/second with 15-second blocking
- Public tokens: 25 requests/second with 15-second blocking
- Premium tokens: 30 requests/second with 10-second blocking
- Admin tokens: 40 requests/second with 5-second blocking

## Prerequisites

- Docker and Docker Compose
- Go 1.21.5 or later (for local development)
- hey tool for load testing (`go install github.com/rakyll/hey@latest`)

## Running with Docker Compose

1. Clone the repository:
```bash
git clone https://github.com/JRRGomes/rate-limiter.git
cd rate-limiter
```

2. Start the application and Redis using Docker Compose:
```bash
docker compose up --build
```

The application will be available at `http://localhost:8080`.

## Testing Rate Limits

You can test different rate limit scenarios using the hey tool:

1. Test IP-based limiting:
```bash
hey -n 30 -c 10 http://localhost:8080/
```

2. Test public token:
```bash
hey -n 35 -c 10 -H "API_KEY: public-token" -H "TOKEN_TYPE: public" http://localhost:8080/
```

3. Test premium token:
```bash
hey -n 40 -c 10 -H "API_KEY: premium-token" -H "TOKEN_TYPE: premium" http://localhost:8080/
```

4. Test admin token:
```bash
hey -n 50 -c 10 -H "API_KEY: admin-token" -H "TOKEN_TYPE: admin" http://localhost:8080/
```

## Automated Tests

There is also an automated test that cover all the rate limiter logics.
You can run the limiter_test.go file inside /limiter folder:

```bash
cd limiter
go test -v
```

## Project Structure

```
rate-limiter/
├── cmd/
│   └── main.go           # Application entry point
├── config/
│   └── config.go         # Configuration management
├── limiter/
│   ├── limiter.go        # Core rate limiting logic
│   ├── limiter_test.go   # Automated tests for rate limiting logic
│   ├── middleware.go     # HTTP middleware
│   ├── redis.go          # Redis storage implementation
│   └── strategy.go       # Storage interface
├── docker-compose.yml    # Docker Compose configuration
├── Dockerfile           # Docker build instructions
└── README.md
```

## Local Development

For local development without Docker:

1. Install and start Redis:
```bash
sudo apt update
sudo apt install redis-server
sudo service redis-server start
```

2. Run the application:
```bash
go run cmd/main.go
```

3. To stop Redis after development:
```bash
sudo service redis-server stop
```

## Clean Up

To stop and remove the Docker containers:
```bash
docker compose down
```

To remove all stored data (including Redis volume):
```bash
docker compose down -v
```

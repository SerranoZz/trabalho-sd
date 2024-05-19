﻿# Trabalhos de Sistemas Distribuídos
## Trabalho 1 - RPC

## Trabalho 2 - Raft
### Configurando ambiente
1. Clone o repositório: <br/>
`$ git clone https://github.com/SerranoZz/trabalho-sd.git`
2. Vá para o diretório: <br/>
`$ cd trabalho-sd/'Trabalho 2 - Raft'`
3. Execute o comando Go: <br/>
`$ go mod init trab-sd`
4. Vá para o diretório: <br/>
`$ cd src/labrpc`
5. Execute o comando Go: <br/>
`$ go mod init labrpc`
6. Vá para o diretório: <br/>
`$ cd ../..`
7. Execute os comandos Go: <br/>
`$ go mod edit -replace=labrpc=".\src\labrpc" && go get labrpc`
8. Vá para o diretório: <br/>
`$ cd src/raft`
9. Execute o comando: <br/>
`$ go test -run 2A`

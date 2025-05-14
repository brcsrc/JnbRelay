# JnbRelay
simple TLS termination proxy in go

### Building
for x86_64/AMD64 Linux, Docker hosts etc
```shell
GOOS=linux GOARCH=amd64 go build -o jnb-relay
```
for ARM64 Linux, Docker hosts etc
```shell
GOOS=linux GOARCH=arm64 go build -o jnb-relay
```
for mac Intel chips
```shell
GOOS=darwin GOARCH=amd64 go build -o jnb-relay
```
for mac M series chips
```shell
GOOS=darwin GOARCH=amd64 go build -o jnb-relay
```
let go build infer your os and arch type
```shell
go build -o jnb-relay
```

### Usage
```shell
./jnb-relay \
  --host 0.0.0.0 \
  --port 443 \
  --proxy-for-host 127.0.0.1 \
  --proxy-for-port 8443 \
  --cert cert.crt \
  --key key.pem
```

### Creating self signed certs with openssl
```shell
openssl req -x509 -newkey rsa:4096 \
    -keyout key.pem \
    -out cert.crt \
    -sha256 -days 3650 -nodes \
    -subj "/C=XX/ST=StateName/L=CityName/O=CompanyName/OU=CompanySectionName/CN=CommonNameOrHostname"
```

### Use Netcat for behavior testing
> tested with BSD netcat

```shell
while true; do echo -ne "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"message\": \"ok\"}" | nc -l 127.0.0.1 8443; done
```

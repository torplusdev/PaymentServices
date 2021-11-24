FROM golang:1.16.8 as build
WORKDIR /opt/torplus/go-payment-service
ARG PG_VERSION
ENV PG_VERSION $PG_VERSION
RUN apt-get update && apt-get install -y build-essential manpages-dev 
COPY ./payment-gateway-webui/go.mod ./payment-gateway-webui/go.mod
COPY ./payment-gateway-webui/go.sum ./payment-gateway-webui/go.sum
COPY ./PaymentGateway/go.mod ./PaymentGateway/go.mod
COPY ./PaymentGateway/go.sum ./PaymentGateway/go.sum
COPY ./ProtocolCustomizations/go.mod ./ProtocolCustomizations/go.mod
RUN cd PaymentGateway && go mod download

COPY . .
WORKDIR /opt/torplus/go-payment-service/PaymentGateway
RUN CGO_ENABLED=1 go build -o ./main_linux  ./cmd/main/ && mv ./main_linux /opt/torplus/main_linux

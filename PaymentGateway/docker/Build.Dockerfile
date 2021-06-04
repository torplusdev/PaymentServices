FROM golang:latest as build
WORKDIR /opt/paidpiper/go-payment-service
RUN apt-get update && apt-get install -y build-essential manpages-dev 
COPY . .
WORKDIR /opt/paidpiper/go-payment-service/PaymentGateway
RUN CGO_ENABLED=1 go build -o ./main_linux  ./cmd/main/ && mv ./main_linux /opt/paidpiper/main_linux
#//TODO COMPLETe

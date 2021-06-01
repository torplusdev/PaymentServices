FROM golang:latest as build
WORKDIR /opt/paidpiper/go-payment-service
RUN apt-get update && apt-get install -y build-essential manpages-dev 
# COPY PaymentGateway/go.mod,PaymentGateway/go.sum PaymentGateway/
# COPY PaymentGateway/go.mod,PaymentGateway/go.sum PaymentGateway/

#RUN cd PaymentGateway && go mod download

COPY . .
WORKDIR /opt/paidpiper/go-payment-service/PaymentGateway
RUN CGO_ENABLED=1 go build -o ./main_linux  ./cmd/main/ && mv ./main_linux /opt/paidpiper/main_linux

FROM torplusserviceregistry.azurecr.io/private/tor
RUN apt-get update && \
    apt-get install -y curl gettext-base supervisor && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /opt/paidpiper/
ENV seed=SAGWKBJQ5J2AJAM5YTEUGSOEXUBOVNZJ6FNJDN7CBKZ543VRQ7XTDFIQ
ENV no_conf=0
COPY --from=build  /opt/paidpiper/main_linux payment-gateway
COPY PaymentGateway/config.json config.json
COPY PaymentGateway/docker/config.json.tmpl config.json.tmpl
COPY PaymentGateway/docker/pg-docker-entrypoint.sh /pg-docker-entrypoint.sh
RUN chmod 755 /pg-docker-entrypoint.sh
COPY PaymentGateway/docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY PaymentGateway/docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY PaymentGateway/docker/client_stage_tor_rc /client_stage_tor_rc

ENTRYPOINT ["/usr/bin/supervisord"]

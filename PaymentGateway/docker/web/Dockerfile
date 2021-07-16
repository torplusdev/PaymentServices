FROM pg_build as build

#FROM torplusserviceregistry.azurecr.io/private/tor:latest

FROM torplusserviceregistry.azurecr.io/private/tor_plus:latest
#ENV PP_ENV=stage
ENV PP_ENV=prod

RUN apt-get update && \
    apt-get install -y curl gettext-base supervisor nginx && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /opt/paidpiper/
ENV no_conf=0

ENV role=hs_client
COPY --from=build  /opt/paidpiper/main_linux payment-gateway
COPY PaymentGateway/config.json config.json
COPY PaymentGateway/docker/config.json.tmpl config.json.tmpl
COPY PaymentGateway/docker/pg-docker-entrypoint.sh /pg-docker-entrypoint.sh
RUN chmod 755 /pg-docker-entrypoint.sh
COPY PaymentGateway/docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY PaymentGateway/docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY PaymentGateway/docker/web/nginx.conf /etc/nginx/nginx.conf
COPY PaymentGateway/docker/web/start.sh /start.sh
RUN chmod 755 /start.sh
ENTRYPOINT ["/start.sh"]

FROM pg_build as build
FROM torplusserviceregistry.azurecr.io/private/tor:latest
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

ENTRYPOINT ["/usr/bin/supervisord"]

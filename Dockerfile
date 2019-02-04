FROM bitnami/minideb:stretch

RUN apt-get update && \
    apt-get upgrade -y && \
    install_packages ca-certificates

COPY dnsmetrics /usr/bin/dnsmetrics

CMD ["/usr/bin/dnsmetrics", "-config", "/local/config.yml"]

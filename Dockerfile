FROM golang:1.9 AS builder

WORKDIR /go/src/github.com/justwatchcom/elasticsearch_exporter

COPY config/vendor ./vendor
COPY config/collector ./collector
COPY config/*.go config/Makefile config/VERSION config/.promu.yml ./

RUN make build \
    && cp ./elasticsearch_exporter /elasticsearch_exporter

FROM scratch

# ElasticSearch Exporter image for OpenShift Origin

LABEL io.k8s.description="ElasticSearch Prometheus Exporter." \
      io.k8s.display-name="ElasticSearch Exporter" \
      io.openshift.expose-services="9113:http" \
      io.openshift.tags="elasticsearch,exporter,prometheus" \
      io.openshift.non-scalable="true" \
      help="For more information visit https://github.com/Worteks/docker-esearchexporter" \
      maintainer="Samuel MARTIN MORO <faust64@gmail.com>" \
      version="1.0"

COPY --from=builder /elasticsearch_exporter /elasticsearch_exporter

ENTRYPOINT ["/elasticsearch_exporter"]

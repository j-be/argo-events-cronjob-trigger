FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY cronjob-trigger .
USER 65532:65532

ENTRYPOINT ["/cronjob-trigger"]

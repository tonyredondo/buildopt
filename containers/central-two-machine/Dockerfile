FROM eclipse-temurin@sha256:a5418a1fcf440bb273e1db3bce5b0794eb78bfc9d044ba740de76dcbe6075f50

RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates git jq \
    && rm -rf /var/lib/apt/lists/*

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

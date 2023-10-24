FROM golang AS builder

WORKDIR /build
COPY . .
RUN make build

FROM nginx:alpine as runner

RUN useradd -ms /bin/bash apps -u 1001 -g 1002 -d /app

COPY --chown=apps:apps ./nginx.conf /etc/nginx/nginx.conf
COPY --chown=apps:apps --from=builder /build/inkcrop /app/inkcrop

CMD ["/app/entrypoint.sh"]
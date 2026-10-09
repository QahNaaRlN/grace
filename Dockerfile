FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/grace-api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -o /out/grace-auth-admin ./cmd/auth-admin

FROM alpine:3.22
RUN addgroup -S -g 10001 grace && adduser -S -u 10001 grace -G grace
USER grace
COPY --from=build /out/grace-api /usr/local/bin/grace-api
COPY --from=build /out/grace-auth-admin /usr/local/bin/grace-auth-admin
EXPOSE 8080
ENTRYPOINT ["grace-api"]


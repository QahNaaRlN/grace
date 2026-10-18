FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/grace-api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -o /out/grace-auth-admin ./cmd/auth-admin
RUN CGO_ENABLED=0 go build -trimpath -o /out/grace-statement-admin ./cmd/statement-admin

FROM alpine:3.22
RUN addgroup -S -g 10001 grace && adduser -S -u 10001 grace -G grace
USER grace
COPY --from=build /out/grace-api /usr/local/bin/grace-api
COPY --from=build /out/grace-auth-admin /usr/local/bin/grace-auth-admin
COPY --from=build /out/grace-statement-admin /usr/local/bin/grace-statement-admin
COPY internal/pdfextract/testdata/columns.pdf internal/pdfextract/testdata/cards.pdf /usr/share/grace/statement-qualification/
EXPOSE 8080
ENTRYPOINT ["grace-api"]


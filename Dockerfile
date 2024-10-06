# Temporary builder with npm, etc installed.
FROM golang:1.23 as builder
WORKDIR /app
COPY . ./
# Build JS bundle and replace client.js with it.
# TODO: This feels a bit messy...
RUN apt update && apt install -y npm
RUN cd js && npm ci && npm run build && mv bundle.js client.js
RUN git config --global --add safe.directory /app
RUN go build -v -o app

# Fresh image without npm, for smaller size.
FROM golang:1.23
RUN apt update && apt install -y sqlite3

WORKDIR /app
COPY config.json .
COPY --from=builder /app/app app
ENTRYPOINT ["/app/app"]

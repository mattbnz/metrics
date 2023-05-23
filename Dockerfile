FROM golang:1.20
WORKDIR /app
COPY . ./
RUN apt update && apt install -y sqlite3
RUN git config --global --add safe.directory /app
RUN go build -v -o app
ENTRYPOINT ["/app/app"]

# Official Go image with the complete toolchain.
FROM golang:1.26.3

WORKDIR /app

# Copy dependency manifests first so dependency downloads are cached.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Compile once during image creation without changing the source tree.
RUN go build ./...

CMD ["bash"]

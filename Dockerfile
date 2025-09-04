

# Add the commands needed to put your compiled go binary in the container and
# run it when the container starts.
#
# See https://docs.docker.com/engine/reference/builder/ for a reference of all
# the commands you can use in this file.
#
# In order to use this file together with the docker-compose.yml file in the
# same directory, you need to ensure the image you build gets the name
# "kadlab", which you do by using the following command:
#
# $ docker build . -t kadlab
FROM golang:1.23.5-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o kadlab cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/kadlab .
EXPOSE 4000
ENTRYPOINT ["./kadlab"]
FROM golang:1

ENV PROJECT=public-brands-api
ENV BUILDINFO_PACKAGE="github.com/Financial-Times/service-status-go/buildinfo."

ARG GITHUB_USERNAME
ARG GITHUB_TOKEN

RUN git config --global url."https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@github.com".insteadOf "https://github.com"

COPY . /${PROJECT}/
WORKDIR /${PROJECT}

# Build app
RUN VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && LDFLAGS="-s -w -X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && CGO_ENABLED=0 go build -mod=readonly -a -o /artifacts/${PROJECT} -ldflags="${LDFLAGS}" \
  && echo "Build flags: ${LDFLAGS}"

# Multi-stage build - copy certs and the binary into the image
FROM scratch
WORKDIR /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /artifacts/* /

CMD [ "/public-brands-api" ]

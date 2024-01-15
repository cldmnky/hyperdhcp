# Use distroless as minimal base image to package the zupd binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
#FROM gcr.io/distroless/static:nonroot
FROM registry.access.redhat.com/ubi9/ubi:9.3-1476
ARG TARGETPLATFORM
LABEL PROJECT="hyperdhcp" \
      MAINTAINER="hyperdhcp Authors" \
      DESCRIPTION="hyperdhcp Operator" \
      LICENSE="Apache-2.0" \
      PLATFORM="$TARGETPLATFORM" \
      VCS_URL="github.com/cldmnky/hyperdhcp" \
      COMPONENT="hyperdhcp"
WORKDIR /
COPY ${TARGETPLATFORM}/hyperdhcp /hyperdhcp
USER 65532:65532
ENTRYPOINT ["/hyperdhcp"]

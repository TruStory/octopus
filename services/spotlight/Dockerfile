FROM ubuntu:18.04
RUN apt-get update
RUN apt-get install -y librsvg2-bin
ENV SPOTLIGHT_STORAGE_PATH /usr/spotlight/storage
RUN mkdir -p ${SPOTLIGHT_STORAGE_PATH}
WORKDIR /usr/spotlight
ADD bin/spotlightd  /usr/spotlight/spotlightd
ENTRYPOINT [ "/usr/spotlight/spotlightd" ]
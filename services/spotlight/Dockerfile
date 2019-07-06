FROM surnet/alpine-wkhtmltopdf:3.8-0.12.5-full 

ENV SPOTLIGHT_STORAGE_PATH /usr/spotlight/storage
RUN mkdir -p ${SPOTLIGHT_STORAGE_PATH}
WORKDIR /usr/spotlight
ADD bin/spotlightd  /usr/spotlight/spotlightd
ENTRYPOINT [ "/usr/spotlight/spotlightd" ]
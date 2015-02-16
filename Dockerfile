FROM ubuntu:latest
MAINTAINER aduros <alexandre.duros@canal-plus.com>

WORKDIR /srv

RUN apt-get update && \
    apt-get upgrade --quiet --yes

RUN apt-get install --quiet --yes pkg-config
RUN apt-get install --quiet --yes make
RUN apt-get install --quiet --yes golang
RUN apt-get install --quiet --yes gccgo
RUN apt-get install --quiet --yes libav-tools
RUN apt-get install --quiet --yes libavformat-dev
RUN apt-get install --quiet --yes libjpeg-dev

ADD src /srv/src
ADD configure Makefile Makefile.inc /srv/
RUN mkdir -p .obj
RUN ./configure && make

ADD videos /var/videos
ENV VIDEOS /var/videos

ADD interface /var/www
ENV UI        /var/www

EXPOSE 3000

CMD ./DashMe -video=$VIDEOS -ui=$UI

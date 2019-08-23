FROM arm32v6/alpine

RUN apk update \
	&& apk add alpine-sdk go \
	&& adduser -s /bin/sh builder -G abuild -D \
	&& su - builder


WORKDIR /home/builder
USER builder

RUN abuild-keygen -a -n

ENV GOARM=6
ENV GOARCH=arm

ADD . .
RUN abuild

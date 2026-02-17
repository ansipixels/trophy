FROM scratch
COPY NAME /usr/bin/trophy
ENV HOME=/home/user
ENTRYPOINT ["/usr/bin/trophy"]

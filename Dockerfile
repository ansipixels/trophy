FROM scratch
COPY trophy /usr/bin/trophy
ENV HOME=/home/user
ENTRYPOINT ["/usr/bin/trophy"]

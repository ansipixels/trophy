FROM scratch
# COPY docs/trophy.glb /data/trophy.glb
COPY trophy /usr/bin/trophy
ENV HOME=/data
VOLUME ["/data"]
ENTRYPOINT ["/usr/bin/trophy"]
# CMD ["trophy.glb"]

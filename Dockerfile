FROM scratch
COPY docs/trophy.glb /home/user/trophy.glb
COPY trophy /usr/bin/trophy
ENV HOME=/home/user
ENTRYPOINT ["/usr/bin/trophy"]
CMD ["trophy.glb"]

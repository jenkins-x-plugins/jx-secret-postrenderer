FROM ghcr.io/jenkins-x/jx-boot:latest

ENTRYPOINT ["jx-secret-postrenderer"]

COPY ./build/linux/jx-secret-postrenderer /usr/bin/jx-secret-postrenderer

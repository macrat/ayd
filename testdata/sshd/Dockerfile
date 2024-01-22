FROM alpine:latest

RUN apk add openssh
RUN echo "AcceptEnv ayd_*" >> /etc/ssh/sshd_config

RUN adduser -D foo && echo foo:bar | chpasswd
RUN ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N '' | grep -Eo 'SHA256:[^ ]+' > /home/foo/fingerprint

COPY ./scripts /usr/local/bin
RUN chmod 777 /usr/local/bin/*

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D", "-e"]

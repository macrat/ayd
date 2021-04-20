Ayd? container image
====================

The container image of [Ayd?](https://github.com/macrat/ayd).
This image includes [Ayd?](https://github.com/macrat/ayd), [ayd-mail-alert](https://github.com/macrat/ayd-mail-alert), and [ayd-slack-alert](https://github.com/macrat/ayd-slack-alert).

There is 3 variants of the base images.

- `latest`, `alpine`: Balanced variant. This is tiny but you can use shell.
- `busybox`: Minimal variant. You can use this if you won't use shell.
- `ubuntu`: Large variant. You can use `apt` command for adding command that you want.


## Usage

### Simple usage

Below example is checking `http://your-service.example.com` every 10 minutes.

``` shell
$ docker run -p 9000:9000 macrat/ayd 10m https://your-service.example.com
```

You can see status page on http://localhost:9000/status.html

Please see [Ayd project page](https://github.com/macrat/ayd) for more information.

### Persistence

This container write log to `/var/log/ayd/ayd.log`.
This log is also works as database to restore state when restart.

``` shell
$ docker run -p 9000:9000 -v ./ayd.log:/var/log/ayd/ayd.log macrat/ayd $YOUR_TARGETS
```

### Send alert to e-mail

``` shell
$ docker run -p 9000:9000 \
    -e "smtp_server=$YOUR_SMTL_SERVER" \
    -e "smtp_username=$YOUR_SMTP_USERNAME" \
    -e "smtp_password=$YOUR_SMTP_PASSWORD" \
    -e "ayd_mail_to=$YOUR_EMAIL" \
    macrat/ayd -a exec:ayd-mail-alert $YOUR_TARGETS
```

seealso: [ayd-mail-alert](https://github.com/macrat/ayd-mail-alert)

### Send alert to Slack

``` shell
$ docker run -p 9000:9000 \
    -e "slack_webhook_url=$YOUR_SLACK_WEBHOOK_URL" \
    macrat/ayd -a exec:ayd-slack-alert $YOUR_TARGETS
```

seealso: [ayd-slack-alert](https://github.com/macrat/ayd-slack-alert)

![Ayd?](./assets/logo.svg)

Easiest status monitoring service to check something service is dead or alive.


## Features

- status checking with:
  * HTTP/HTTPS
  * ICMP echo (ping)
  * TCP connect
  * DNS resolve
  * execute external command (or script file)
- view status page in browser, console, or program.
- kick alert if target failure.

### Good at
- Make a status page for temporary usage. (You can start it via one command! And, stop via just Ctrl-C!)
- Make a status page for a minimal system. (Single binary server, single log file, there is no database!)

### Not good at
- Complex customization, extension. (There are nothing options for customizing.)
- Investigate more detail. (This is just for check dead or alive.)


## Quick start

1. Download latest version from [release page](https://github.com/macrat/ayd/releases/).

2. Extract downloaded package and place **ayd** (or **ayd.exe**) to some place.

3. Run the server.

``` shell
$ ayd https://your-service.example.com ping:another-host.example.com
```

4. Check your status page.

- HTML page for browser: [http://localhost:9000/status.html](http://localhost:9000/status.html)
- Plain text page for use in console: [http://localhost:9000/status.txt](http://localhost:9000/status.txt)
- Json format for handling in program: [http://localhost:9000/status.json](http://localhost:9000/status.json)


## Usage detail

### Status page and endpoints

Ayd has these pages/endpoints.

|path                                             |description                                                         |
|-------------------------------------------------|--------------------------------------------------------------------|
|[/status.html](http://localhost:9000/status.html)|Human friendly status page in HTML.                                 |
|[/status.txt](http://localhost:9000/status.txt)  |Human friendly status page in plain text.                           |
|[/status.json](http://localhost:9000/status.json)|Machine readable status page in JSON format.                        |
|[/metrics](http://localhost:9000/metrics)        |Minimal status page for use by [Prometheus](https://prometheus.io/).|
|[/healthz](http://localhost:9000/healthz)        |Health status page for checking status of Ayd itself.               |


### Specify target

Ayd demands URI as targets.
Please see below what you can use as a scheme (protocol).

#### http / https

Fetch HTTP(S) page and check status code is 2xx or not.

You can use GET, HEAD, POST, and OPTIONS method by specifying like `http-post://...` or `https-head://...`.
The default method is GET.

Ayd will Follow redirect maximum 10 times.

examples:
- `http://example.com`
- `https://example.com`
- `http-head://example.com/path/to/somewhere`
- `https-options://example.com/abc?def=ghi`

#### ping

Send ICMP echo request (a.k.a. ping command) and check the server is connected or not.

Ayd sends 4 packets in 2 seconds and expects all packets to return.

examples:
- `ping:example.com`
- `ping:192.168.1.1`

#### tcp

Connect to TCP and check the service listening or not.

examples:
- `tcp:example.com:3309`

#### dns

Resolve hostname via DNS and check the host exists or not.

examples:
- `dns:example.com`

#### exec

Execute external command and check return code is 0 or not.

The command's stdout and stderr will be captured as a message of the status check record.
You should keep output as short as possible because Ayd is not good at record a long message.

You can specify the first argument as the fragment of URI like below.

```
exec:/path/to/command#this-is-argument
```

Above target URI works the same as the below command in the shell.

``` shell
$ /path/to/command this-is-argument
```

And, you can specify environment arguments as the query of URI like below.

```
exec:/path/to/command?something=foobar&hello=world
```

Above target URI works the same as the below command in the shell.

```
$ export something=foobar
$ export hello=world
$ /path/to/command
```

examples:
- `exec:./check.exe`
- `exec:/usr/local/bin/check.sh`

#### source

This is a special scheme for load targets from a file.
Load each line in the file as a target URI and check all targets.

The line that starts with `#` will ignore as a comment.

examples:
- `source:./targets.txt`
- `source:/path/to/targets.txt`


### Specify check interval/schedule

In default, Ayd will check targets every 5 minutes.

You can place the timing specification before the target specification like below if you want.

``` shell
$ ayd 10m https://your-service.example.com 1h https://another-service.example.com
```

The above command will check `your-service.example.com` every 10 minutes, and check `another-service.example.com` every 1 hour.

You can also use [the Cron](https://en.wikipedia.org/wiki/Cron) style spec as a timing spec like below.

``` shell
$ ayd '*/5 6-21 * * *' https://your-service.example.com https://another-service.example.com
```

The above command will check `your-service.example.com` and `another-service.example.com` every 5 minutes from 6 a.m. to 9 p.m.


### Change log place

Ayd will save the log file in the current directory in default.
You can change this with `-o` option like below.

``` shell
$ ayd -o /path/to/ayd.log ping:example.com
```

There is no feature to log rotate.
Please consider using the log rotation tool if you have a plan to use it for a long time.
(Ayd can handle the huge log, but it is not easy to investigate the huge log when trouble)


### Setup alerting

Ayd can kick a URI when a target status checks failure.
You may want to use exec or HTTP for alerting.
(Even you can use ping, DNS, etc as alerting. but... it's useless in almost all cases)

Ayd will kick alert at only the timing that incident caused, and it won't kick at the timing that continuing or resolved the incident.

You can specify alerting URI like below.

``` shell
$ ayd -a https://alert.example.com/alert https://target.example.com
```

In the above example, Ayd access `https://alert.example/alert` with the below queries when `https://target.example.com` down.

|query name      |example                     |description                  |
|----------------|----------------------------|-----------------------------|
|`ayd_target`    |`https://target.example.com`|The alerting target URI      |
|`ayd_checked_at`|`2001-02-03T16:05:06+09:00` |The checked timestamp        |
|`ayd_status`    |`FAILURE` or `UNKNOWN`      |The status of target checking|

If you want to send an email via SMTP as an alert, you can use [ayd-mail-alert](https://github.com/macrat/ayd-mail-alert).
Please download from [release page of ayd-mail-alert](https://github.com/macrat/ayd-mail-alert/releases) and use like below.

``` shell
$ export SMTP_SERVER=smtp.example.com:465 SMTP_USERNAME=your-name SMTP_PASSWORD=your-password
$ export AYD_MAIL_TO="your name <your-email@example.com>"

$ ayd -a exec:ayd-mail-alert https://target.example.com
```

Please see more information in [the readme of ayd-mail-alert](https://github.com/macrat/ayd-mail-alert#readme).


### Change listen port

You can change the HTTP server listen port with `-p` option.
In default, Ayd uses port 9000.


### Check status just once

If you want to use Ayd in a script, you may use `-1` option.
Ayd will check status just once and exit when passed `-1` option.

Exit status code is 0 if all targets are healthy.
If some targets are unhealthy, the status code will 1.
And, if your arguments are wrong (or can't resolve hostnames, or exec scripts not found), the status code will 2.

Daemonize SDNotify
==================
Small utility to convert a blocking sdnotify program into one that
daemonizes itself (kind of acting the same as systemd would)

Why?
* Running SDNotify-programs on BSDs (where the process daemonizes itself to the background)
* On test environments

How it works
```bash
$ cd testdaemon/
$ go build
$ cd ..
$ go build && ./sdnotify-wrapper derp testdaemon/testdaemon
2023/08/08 11:32:19 wait till SIGINT|SIGTERM
$ ps aux | grep testdaemon
me             65260   0.0  0.0  4399296    728 s000  S+   11:32AM   0:00.00 grep testdaemon
me             65245   0.0  0.0  4982348   2328 s000  S    11:32AM   0:00.00 testdaemon/testdaemon
$ kill 65245
2023/08/08 11:32:30 wait done
```

Inspired on:
https://github.com/coreos/sdnotify-proxy

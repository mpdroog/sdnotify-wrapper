Daemonize SDNotify
==================
Convert SDNotify-signal of systemd into a fork.

Why is this useful?

* Running SDNotify-programs on BSDs;
* On test environments on non-Linux machines;

How it works
```bash
$ cd testdaemon/
$ go build
$ cd ..
$ go build && ./sdnotify-wrapper derp.sock testdaemon/testdaemon
2023/08/08 11:32:19 wait till SIGINT|SIGTERM
$ ps aux | grep testdaemon
me             65260   0.0  0.0  4399296    728 s000  S+   11:32AM   0:00.00 grep testdaemon
me             65245   0.0  0.0  4982348   2328 s000  S    11:32AM   0:00.00 testdaemon/testdaemon
cat derp.sock.pid
65245
$ kill 65245
2023/08/08 11:32:30 wait done
```

Why prefer SDNotify over forking in Go?

* sdnotify allows standard Go cross-compiling for your daemon
* No CGO needed
* Ugly forking isolated in this small tool for non-Linux

Example SDNotify code
```go
import "github.com/coreos/go-systemd/daemon"
func main() {
    ..init code here..

	sent, e := daemon.SdNotify(false, "READY=1")
	if e != nil {
		panic(e)
	}
	if !sent {
		panic("SystemD notify NOT sent\n")
	}

	..sleep code here..
}
```

Your service file
```
[Service]
Type=notify
```

Inspired on:
https://github.com/coreos/sdnotify-proxy

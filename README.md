# cronmutex

[![Build Status](https://travis-ci.org/emgag/cronmutex.svg?branch=master)](https://travis-ci.org/emgag/cronmutex)
[![Go Report Card](https://goreportcard.com/badge/github.com/emgag/cronmutex)](https://goreportcard.com/report/github.com/emgag/cronmutex)

**BETA**: feature complete, but not used in production yet.

## Usage

```
Usage:
  cronmutex [flags] <MUTEX-NAME> <COMMAND>

Flags:
  -c, --config string       config file (default is /etc/cronmutex.yml)
  -f, --fire-n-forget       Don't hold (extend) the lock while the command is running
  -h, --help                help for cronmutex
  -m, --mutex-ttl int       The TTL of the lock in X seconds
  -n, --noout               Don't dump STDOUT and STDERR from command
  -w, --random-wait int32   Wait for a random duration between 0 and X seconds before acquiring the lock and starting the command
  -t, --ttl int             Kill command after X seconds. Default is to wait until the command finishes by itself
  -v, --verbose             Tell what's happening with cronmutex
      --version             Print version and exit
```

* `--config` Set path to config file instead of using the default */etc/cronmutex.yml* 

* `--fire-n-forget` Don't hold (extend) the lock while the command is running. Just keep lock until the lock's TTL (`--mutex-ttl`) expires and continue running the command without holding the lock (unless a `--ttl` is set and the command gets killed before).

* `--mutex-ttl X` The TTL of the lock in X seconds. Note that unless `--fire-n-forget` is set as well, the lock keeps getting extended by this amount before it expires until the command finishes and the lock gets released.

* `--noout` Don't dump STDOUT and STDERR from command.

* `--random-wait X` Wait for a random duration between 0 and X seconds before acquiring the lock and starting the command.  

* `--ttl X` Kill command after X seconds. Default is to wait until the command finishes by itself.  

* `--verbose` Tell what's happening with cronmutex.
 
* `--version` Print version and exit

## Configuration

Unless overwritten by the `--config` option, cronmutex looks for a [cronmutex.yml](/cronmutex.yml.dist) in */etc*, *$HOME/.config* and current working directory.

Example config:

```
redis:
  uri: redis://127.0.0.1:6379
  #password: thepasswordifneeeded
mutex:
  prefix: EXAMPLEPREFIX.
  defaultTTL: 300
```

## Build

On Linux:

```
$ mkdir cronmutex && cd cronmutex
$ export GOPATH=$PWD
$ go get -d github.com/emgag/cronmutex
$ cd src/github.com/emgag/cronmutex
$ dep ensure -update
$ make install
```

will download the source and builds binary called _cronmutex_ in $GOPATH/bin.

## License

cronmutex is licensed under the [MIT License](http://opensource.org/licenses/MIT).

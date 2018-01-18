# cronmutex


## Usage

* `--fire-n-forget` Don't hold (extend) the lock while the command is running. Just keep lock until the lock's TTL (`--mutex-ttl`) expires and continue running the command without holding the lock (unless a `--ttl` is set and the command gets killed before).

* `--mutex-ttl X` The TTL of the lock in X seconds. Note that unless `--fire-n-forget` is set as well, the lock keeps getting extended by this amount before it expires until the command finishes and the lock gets released.

* `--noout` Don't dump STDOUT and STDERR from command.

* `--random-wait X` Wait for a random time between 0 and X seconds before acquiring the lock and starting the command.  

* `--ttl X` Kill command after X seconds. Default is to wait until the command finishes by itself.  

* `--verbose` Tell what's happening with cronmutex. 


## Build

TBD

## License

cronmutex is licensed under the [MIT License](http://opensource.org/licenses/MIT).

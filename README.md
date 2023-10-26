# gocrontab

Timed task management similar to Linux crond

## Install

```sh
go install github.com/mengdu/gocrontab/cmd/gocrond@xxx
go install github.com/mengdu/gocrontab/cmd/gocron@xxx
```

## Usage

**1、Config crontab**

`demo.crontab`

```cron
# say hello
* * * * * echo "Hello World!"
# print date
*/2 * * * * echo "Hi! at:$(date +%FT%T%z)"
```

**2、Start `gocrond`**

```sh
gocrond -c demo.crontab
```

**3、Usage cli**

```sh
gocron ls # show list of jobs
gocron exec <id> # manually executing jobs
```

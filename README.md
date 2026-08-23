# logfmt-lint

A lot of application logs are written in logfmt: space-separated
`key=value` pairs, the format logrus, zap's logfmt encoder, and Heroku's
router all produce. It's easy to read by eye but easy to get subtly wrong
in code that emits it — an unterminated quote, a field written twice, a
value that swallowed the next key because a quote wasn't closed. This
tool checks that a log stream is actually well-formed logfmt and prints it
in a form that's easier to scan than the raw line.

## Usage

Build it:

```
go build -o logfmt-lint .
```

Check a file:

```
./logfmt-lint app.log
```

Check several files in one run:

```
./logfmt-lint app.log app.log.1
```

Read from stdin — this is also what happens if you pass no arguments at
all, or pass `-` explicitly:

```
tail -f app.log | ./logfmt-lint
```

Given a line like:

```
ts=2026-08-24T10:03:12Z level=error msg="connection refused" host=db-1 retries=3
```

it prints:

```
stdin:1
  ts      = 2026-08-24T10:03:12Z
  level   = error
  msg     = connection refused
  host    = db-1
  retries = 3
```

Lines that don't parse (an unterminated quote, an empty key) or that fail
validation (a key repeated within the same line) are reported to stderr
with the source name and line number, and the process exits with status 1
if anything in the run failed.

## Status

Early. Parsing and validation currently cover syntax only — there's no way
yet to require that specific fields such as `level` or `msg` be present.
Read the source for the exact rules; it's short.

## License

MIT, see LICENSE.

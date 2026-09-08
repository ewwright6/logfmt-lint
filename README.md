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

Require certain fields to be present with `-require`:

```
./logfmt-lint -require=level,msg app.log
```

A line missing any of the listed keys is reported the same way as a
duplicate-key error, with the missing key named.

Some logfmt producers write bare keys — a token with no `=` at all, like
`debug` in `level=error debug` — which this tool normally accepts as a
key with an empty value. Reject them instead with `-strict`:

```
./logfmt-lint -strict app.log
```

A bare key is reported the same way as a duplicate-key error.

When stdout is a terminal, the header and field keys are printed in color.
Piping or redirecting output turns this off automatically, and setting
`NO_COLOR` (to any non-empty value) turns it off regardless of where
output is going.

Print JSON instead of the aligned text form with `-json`, one JSON object
per line:

```
./logfmt-lint -json app.log
```

```
{"source":"app.log","line":1,"fields":{"host":"db-1","level":"error","msg":"connection refused","retries":"3","ts":"2026-08-24T10:03:12Z"}}
```

Gzip-compressed input is decompressed automatically, whether it's a file
or piped in over stdin — detection is by magic number, not filename, so
a `.gz` extension isn't required:

```
./logfmt-lint app.log.gz
zcat app.log.gz | ./logfmt-lint
```

## Status

Early. Read the source for the exact rules; it's short.

## License

MIT, see LICENSE.

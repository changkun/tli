# tli

A [Things](https://culturedcode.com/things/) CLI.

> **_Caution: non-stable implementation, use at your own risk._**

```
go get changkun.de/x/tli
```

## Enable Mail to Things

See https://culturedcode.com/things/support/articles/2908262/.

Limitations on Things:

- There is no way, via email, to set tags or any other metadata.
- Content is limited than 2,000 characters, otherwise it will be truncated.
- Only supports plain text, also means no attachments.
- Maximum of 50 emails per 24-hour period.

## Features

- `tli` caches your TODOs
- `tli` splits your content to avoid silently truncated content by Things

## Usage

Initialize your environment:

```sh
$ tli init
```

Daily usage:

```sh
$ tli todo [title]
> content body line
> content body line 2
> # enpty line
DONE!

$ tli log
$ tli log 2
```

Configurations and cache files:

- `~/.tli_config`: for configurations
- `~/.tli_history`: for historical retrival

Details:

```
tli is a Linux CLI that supports send items to the Things' Inbox safely.
Specifically, it will save the sent TODO log to prevent if you send too
much to the Things' server. tli also checks your content to make sure your
inputs won't be too large so that the content is not silently truncated
by Things.

BuildVersion: b37519b
BuildTime:    2020-11-15T02:14:51+0100
GoVersion:    go1.15.5

Usage:
  tli [command]

Available Commands:
  help        Help about any command
  init        initialize tli settings
  log         print logs
  todo        create a todo and send it to the Things' Inbox

Flags:
  -h, --help   help for tli

Use "tli [command] --help" for more information about a command.
```

## License

GNU GPL-3.0 Copyright &copy; 2020 [Changkun Ou](https://changkun.de)
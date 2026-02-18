# escapes

Render CP437-encoded ANSI art to UTF-8 terminals.

[![Go Reference](https://pkg.go.dev/badge/github.com/tetsuo/escapes.svg)](https://pkg.go.dev/github.com/tetsuo/escapes)

## CLI usage

Install the `escapes` CLI tool:

```bash
go install github.com/tetsuo/escapes/cmd/escapes@latest
```

```bash
escapes -i art.ans
```

or you can pipe into it:

```bash
escapes < art.ans
```
## Resources

You can find cool ANSI art at:

- [roy/sac](https://www.roysac.com/roy_ansishow.html)
- [16colo.rs](https://16colo.rs/)

![escapes screenshot](screenshot.png)

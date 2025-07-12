# lockstep

[![Go Reference](https://pkg.go.dev/badge/github.com/dcaiafa/lockstep.svg)](https://pkg.go.dev/github.com/dcaiafa/lockstep)

A Go testing primitive for facilitating the testing of complex concurrent systems.

LockStep supports two operations: Emit and Wait. An Emit operation with a
message x will block until a corresponding Wait operation with message x is
processed. Likewise, a Wait operation with y will also block until the
corresponding Emit operation with y is processed.

Example:

```go
func TestExample(t *testing.T) {
  ls := lockstep.New(t)

  const d = 300 * time.Millisecond
  go func() {
    ls.Wait("go1")
    time.AfterFunc(d, func() {
      ls.Emit("done1")
    })
  }()
  go func() {
    ls.Wait("go2")
    <-time.After(d)
    ls.Emit("done2")
  }()

  begin := time.Now()
  ls.Emit("go1")
  ls.Emit("go2")
  ls.Wait("done1", "done2")
  dur := time.Since(begin)

  delta := dur - d
  if delta < 0 {
    delta *= -1
  }
  const e = 50 * time.Millisecond
  if delta > e {
    t.Fatalf("Expected callback in %v, actual was %v", d, dur)
  }
}
```

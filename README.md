# netlimit 🧙🏻‍♂️

netlimit is a small package that allows you to control bandwitdh usage of `net.Listener` and `net.Conn`, it delivers custom wrapper types around `net.Listener` and `net.Conn` interfaces and util functions like `netlimit.Listen()` and `netlimit.ListenCtx()` to bootstrap the whole process

---

# Usage

Create `netlimit.Listener`

```
//globalLimit limits bandwidth of a listener
globalLimit := 1024 //Bps

//localLimit limits bandwidth of a single connection
localLimit := 512 //Bps

ln, err := netlisten.Listen(proto, addr, globalLimit, localLimit)
```

Use it as you would any other `net.Listener` e.g

```
http.Serve(ln, handler)
```

You can tweak limits during runtime

Change local(per connection) limit use

```
err := ln.SetLocalLimit(newLocalLimit)
```

Change global(server) limit use

```
err := ln.SetGlobalLimit(newLocalLimit)
```

---
# Resources
https://pkg.go.dev/github.com/charconstpointer/netlimit

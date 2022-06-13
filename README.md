# netlimit  
#### netlimit is a package that allows to control the bandwidth of the net.Conn connections and the limiter itself.
---
### Usage
`netlimit.Listener` satisfies `net.Listener`

Create `netlimit.Listener`
```
//globalLimit limits bandwidth of a listener
globalLimit := 1024 //Bps

//localLimit limits bandwidth of a single connection
localLimit := 512 //Bps

ln, err := netlisten.Listen(proto, addr, globalLimit, localLimit)
```

---
### Docs
`https://pkg.go.dev/github.com/charconstpointer/netlimit`


```
 Listener
 ┌───────────────────────────────┐    updates limits
 │                               ├─────────────────────────────────┐
 │                               │                                 │
 │                               │                                 │
 │                               │                                 │
 │                               │               global limiter    │
 │                               │               ┌─────────────────▼─┐
 └──────────────┬────────────────┘               │                   │
                │                                │                   │
                │                                │                   │
                │                                └─────────────────▲─┘
                │                                                  │
                │                                                  │
                │                                                  │
  net.Conn      │                     Allocator                    │
  ┌─────────────▼───┐                 ┌────────────────┐           │
  │                 │                 │                │           │
  │                 ├──┐              │                │           │
  │                 │  │              │ local limiter  │           │
  │                 │  ├─┐            │ ┌────────────┐ │           │
  │                 │  │ │requests    │ │            │ │           │
  │                 │  │ ├────────────► └────────────┘ ├───────────┘
  │                 │  │ │bandwitdh   │                │   allocates
  └────┬────────────┘  │ │            └────────────────┘   bandwidth
       │               │ │
       └───┬───────────┘ │
           │             │
           └─────────────┘
```

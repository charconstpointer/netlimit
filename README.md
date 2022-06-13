# netlimit  
#### netlimit is a package that allows to control the bandwidth of the net.Conn connections and the limiter itself.
---
### docs
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

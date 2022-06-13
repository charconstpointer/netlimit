// netlimit is a package that allows to control the bandwidth of the net.Conn connections and the limiter itself.
// Below is a simplified architecture diagram:
//
//
// Listener
// ┌───────────────────────────────┐    updates limits
// │                               ├─────────────────────────────────┐
// │                               │                                 │
// │                               │                                 │
// │                               │                                 │
// │                               │               global limiter    │
// │                               │               ┌─────────────────▼─┐
// └──────────────┬────────────────┘               │                   │
//                │                                │                   │
//                │                                │                   │
//                │                                └─────────────────▲─┘
//                │                                                  │
//                │                                                  │
//                │                                                  │
//  net.Conn      │                     Allocator                    │
//  ┌─────────────▼───┐                 ┌────────────────┐           │
//  │                 │                 │                │           │
//  │                 ├──┐              │                │           │
//  │                 │  │              │ local limiter  │           │
//  │                 │  ├─┐            │ ┌────────────┐ │           │
//  │                 │  │ │requests    │ │            │ │           │
//  │                 │  │ ├────────────► └────────────┘ ├───────────┘
//  │                 │  │ │bandwitdh   │                │   allocates
//  └────┬────────────┘  │ │            └────────────────┘   bandwidth
//       │               │ │
//       └───┬───────────┘ │
//           │             │
//           └─────────────┘
package netlimit

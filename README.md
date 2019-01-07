# Bounded Goroutine Pool
`bounded` is a thin concurrency wrapper that provides bounded goroutine management.

Go was designed to have lightweight concurrency primitives with goroutines so that concurrent tasks could be called with `go`.
Unfortunately this is usually taken too far.
Much programs end up having boundless concurrency and as a result they comes with producing a significant amount of overhead.

`bounded` tackles this problem by providing a thin wrapper to allow Go programmers to continue thinking of concurrent code and execution without the worry that they're program is going to produce too much goroutines.

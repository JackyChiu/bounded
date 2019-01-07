# Bounded Goroutine Pool
`bound` is a thin concurrency wrapper that provides bounded goroutine management.

`bound` is a package that provides simple bounded concurrency limits for Go.
Go was designed to have lightweight concurrency primitives with goroutines so that concurrent tasks could be called with `go`.
Unfortunately this is usually taken too far.
Much programs end up having boundless concurrency and as a result they comes with producing a significant amount of overhead.

`bound` tackles this problem by providing a thin wrapper to allow Go programmers to continue thinking of concurrent code and execution without the worry that they're program is going to produce too much goroutines.

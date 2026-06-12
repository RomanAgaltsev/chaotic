package fault

// Kind classifies a Fault for introspection (linting, observability). It does
// not affect behavior.
type Kind int

// Fault kinds, one per built-in fault type.
const (
	KindUnknown Kind = iota
	KindLatency
	KindJittered
	KindError
	KindPanic
	KindConnDrop
	KindHTTPStatus
	KindHeader
	KindClock
	KindDisconnect
)

// Kinded is implemented by every built-in fault so tools can classify it
// without type-switching on unexported types.
type Kinded interface {
	Kind() Kind
}

// KindOf returns f's Kind, or KindUnknown for a custom fault that does not
// implement Kinded.
func KindOf(f Fault) Kind {
	if k, ok := f.(Kinded); ok {
		return k.Kind()
	}
	return KindUnknown
}

package gchain

type sizeHintKind uint8

const (
	sizeHintUnknown sizeHintKind = iota
	sizeHintExact
	sizeHintUpperBound
)

// sizeHint 描述 sequence 的元素数量信息，只用于内部 collector 预分配。
// upper bound 不直接用于 make，避免 Filter 场景为大量被丢弃的元素预分配。
type sizeHint struct {
	kind  sizeHintKind
	count int
}

func exactSizeHint(count int) sizeHint {
	return sizeHint{
		kind:  sizeHintExact,
		count: count,
	}
}

func upperBoundSizeHint(count int) sizeHint {
	if count == 0 {
		return exactSizeHint(0)
	}
	return sizeHint{
		kind:  sizeHintUpperBound,
		count: count,
	}
}

func (h sizeHint) exactCapacity() int {
	if h.kind != sizeHintExact {
		return 0
	}
	return h.count
}

func (h sizeHint) asUpperBound() sizeHint {
	switch h.kind {
	case sizeHintExact, sizeHintUpperBound:
		return upperBoundSizeHint(h.count)
	default:
		return sizeHint{}
	}
}

func (h sizeHint) take(count int) sizeHint {
	if count == 0 {
		return exactSizeHint(0)
	}

	switch h.kind {
	case sizeHintExact:
		return exactSizeHint(min(h.count, count))
	case sizeHintUpperBound:
		return upperBoundSizeHint(min(h.count, count))
	default:
		return upperBoundSizeHint(count)
	}
}

func (h sizeHint) drop(count int) sizeHint {
	switch h.kind {
	case sizeHintExact:
		return exactSizeHint(max(h.count-count, 0))
	case sizeHintUpperBound:
		return upperBoundSizeHint(max(h.count-count, 0))
	default:
		return sizeHint{}
	}
}

func (h sizeHint) chunk(size int) sizeHint {
	chunks := h.count / size
	if h.count%size != 0 {
		chunks++
	}

	switch h.kind {
	case sizeHintExact:
		return exactSizeHint(chunks)
	case sizeHintUpperBound:
		return upperBoundSizeHint(chunks)
	default:
		return sizeHint{}
	}
}

func (h sizeHint) concat(other sizeHint) sizeHint {
	if h.kind == sizeHintUnknown || other.kind == sizeHintUnknown {
		return sizeHint{}
	}

	count := h.count + other.count
	if h.kind == sizeHintExact && other.kind == sizeHintExact {
		return exactSizeHint(count)
	}
	return upperBoundSizeHint(count)
}

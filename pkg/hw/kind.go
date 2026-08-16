//
// Created by Javid Rzayev on 10.08.26.
//

package hw

type Kind uint8

const (
	KindUnknown Kind = iota
	KindSubmit
	KindAlloc
	KindWait
)

func (k Kind) String() string {
	switch k {
	case KindSubmit:
		return "submit"
	case KindAlloc:
		return "alloc"
	case KindWait:
		return "wait"
	}
	return "unknown"
}

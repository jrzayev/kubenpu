//
// Created by Javid Rzayev on 10.08.26.
//

package hw

type Kind int


const (
	KindUnknown Kind = iota // Was added specially if somehow we will not check and we will get an unknown kind
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

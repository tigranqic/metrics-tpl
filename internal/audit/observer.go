package audit

type Observer interface {
	Notify(Event) error
}

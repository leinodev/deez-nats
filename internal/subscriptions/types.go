package subscriptions

type Subscription interface {
	Drain() error
	Unsubscribe() error
}

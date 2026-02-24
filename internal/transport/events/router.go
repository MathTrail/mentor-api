package events

// TODO: implement EventRouter — registers all topic subscribers.
// Mirrors internal/transport/http/router.go for the event transport layer.
// In container.go it will sit alongside the HTTP router:
//
//	httpRouter  := httpserver.NewRouter(...)
//	eventRouter := events.NewEventRouter(...)

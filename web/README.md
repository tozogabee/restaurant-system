# web

The frontend. Framework not chosen yet.

It talks only to the gateway's REST API — never to an individual service, and
never to Postgres, Redis or a message broker directly. The error shape it has to handle is
`httpx.ErrorBody` in `pkg/httpx/response.go`, which every service returns.

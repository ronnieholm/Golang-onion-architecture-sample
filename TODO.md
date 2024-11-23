# TODO

- Add GitHub Actions
- Implementing Robust Transaction Handling in Go (in middleware)
  - https://www.youtube.com/watch?v=7RKOtJ9YB_s
-  Building Secure Go Systems: Key Management, Middleware, and Error Handling (auth, autn middleware)
  - https://www.youtube.com/watch?v=YVT_QHMFFyo
- How to make handlers return error
  - https://www.youtube.com/watch?v=8eZZEEEuudo
- Error handling middleware
  - https://www.youtube.com/watch?v=HF3scbk7a_Q
- Create .env file
- Create assert.Never(...)
- Use yaml as config format: https://github.com/andrearaponi/dito
- Assert naming and param inspiration: https://github.com/flyingmutant/rapid/blob/master/floats.go#L46 and https://github.com/google/go-cmp/blob/master/cmp/report.go#L50
- Pre and post condition checking: https://github.com/pulumi/pulumi/blob/master/pkg/engine/update.go#L657
- Golang 1.22: New Routing Features Eliminate the Need for Third-Party Lib: https://www.reddit.com/r/golang/comments/1ednakn/golang_122_new_routing_features_eliminate_the
- use %q: https://google.github.io/styleguide/go/decisions#use-q
- For example, in HTTP servers, the http.Request.Context method returns a
  context associated with the request. That context is canceled if the HTTP
  client disconnects or cancels the HTTP request (possible in HTTP/2). Passing
  an HTTP request’s context to QueryWithTimeout above would cause the database
  query to stop early either if the overall HTTP request was canceled or if the
  query took more than five seconds.
- Use golangci-lint from makefile
- join errors into aggregate error: https://github.com/golang/go/blob/master/src/errors/join.go#L19
- Make id's on entity comparable? https://go.dev/tour/generics/1
- Adhere to Go decisions: https://google.github.io/styleguide/go/decisions

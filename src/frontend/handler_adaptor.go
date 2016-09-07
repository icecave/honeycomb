package frontend

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/name"
	"github.com/icecave/honeycomb/src/request"
)

// HandlerAdaptor is an http.Handler that forwards to a Honeycomb request.Handler.
type HandlerAdaptor struct {
	Locator     backend.Locator
	Handler     request.Handler
	Interceptor Interceptor
	Logger      *log.Logger
}

func (adaptor *HandlerAdaptor) ServeHTTP(wri http.ResponseWriter, req *http.Request) {
	txn := request.NewTransaction(wri, req)
	txn.Open()
	defer adaptor.close(txn)

	if txn.Error == nil {
		txn.Endpoint = adaptor.Locator.Locate(
			txn.Request.Context(),
			txn.ServerName,
		)
		if txn.Endpoint == nil {
			txn.Error = errors.New("can not locate back-end")
		}
	}

	if adaptor.Interceptor != nil {
		adaptor.Interceptor.Intercept(txn)
	}

	// Forward the request to the normal handler only if the interceptor did not
	// already respond to it ...
	if txn.State == request.StateReceived {
		adaptor.Handler.Serve(txn)
	}
}

// IsRecognised returns true if the given server name name can be handled by
// this handler.
func (adaptor *HandlerAdaptor) IsRecognised(ctx context.Context, serverName name.ServerName) bool {
	return adaptor.Interceptor.Provides(serverName) ||
		adaptor.Locator.Locate(ctx, serverName) != nil
}

func (adaptor *HandlerAdaptor) close(txn *request.Transaction) {
	txn.Close()
	if txn.IsLogged {
		adaptor.Logger.Println(txn)
	}
}

package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/tonyredondo/buildopt/internal/sessioningest"
)

const (
	serverURLEnvironment     = sessioningest.ServerURLEnvironment
	serverTokenEnvironment   = sessioningest.ServerTokenEnvironment
	exportContextEnvironment = sessioningest.ExportContextEnvironment
)

func (gateway *localGateway) deliverSession(
	ctx context.Context,
	client *sessioningest.Client,
	record sessioningest.Record,
) (sessioningest.PutResult, error) {
	if record.GatewayConnectionGeneration != gateway.generation {
		return 0, errors.New(
			"session does not belong to the active gateway generation",
		)
	}
	return client.Deliver(ctx, record)
}

func reportSessionIngest(
	sessionID string,
	result sessioningest.PutResult,
	err error,
	stderr io.Writer,
) {
	if err != nil {
		_, _ = fmt.Fprintf(
			stderr,
			"buildopt: buildopt-server session ingest unavailable: %v\n",
			err,
		)
		return
	}

	action := "accepted"
	if result == sessioningest.PutDuplicate {
		action = "deduplicated"
	}
	_, _ = fmt.Fprintf(
		stderr,
		"buildopt: buildopt-server %s session %s\n",
		action,
		sessionID,
	)
}

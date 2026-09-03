package sharedcache

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// WCNCPActor is the product-state write authority refined beyond transport
// capabilities. Transport may reuse STATE_READ/STATE_WRITE, but every WCNCP
// mutation additionally requires the exact actor grant below. An
// implementation that gives every state writer implicit acceptance authority
// fails the experiment.
type WCNCPActor string

const (
	// WCNCPActorDeveloper is untrusted by default: cache read and local
	// observation only, no control-plane publication.
	WCNCPActorDeveloper WCNCPActor = "DEVELOPER"
	// WCNCPActorTrustedObserver publishes observations for one repository.
	WCNCPActorTrustedObserver WCNCPActor = "TRUSTED_OBSERVER"
	// WCNCPActorValidator reads observations, claims proposals, and publishes
	// validations. It cannot decide.
	WCNCPActorValidator WCNCPActor = "VALIDATOR"
	// WCNCPActorOwner reads proposals and publishes owner decisions. Acceptance
	// never implies source application, commit, push, PR, merge, or production.
	WCNCPActorOwner WCNCPActor = "OWNER"
	// WCNCPActorAdmin manages tokens and revocation. It has no implicit owner
	// decision authority.
	WCNCPActorAdmin WCNCPActor = "ADMIN"
)

// WCNCPGrant is the durable actor refinement for one central token. Fork
// runners are read-only by default; fork_write_allowed is always false in the
// POC and the column exists only to make that invariant explicit in SQL.
type WCNCPGrant struct {
	TokenID          string
	Actor            WCNCPActor
	ObservationWrite bool
	ProposalClaim    bool
	ValidationWrite  bool
	DecisionWrite    bool
}

// WCNCPOperation is one product-state action requiring actor authorization.
type WCNCPOperation string

const (
	WCNCPOpObservationWrite WCNCPOperation = "OBSERVATION_WRITE"
	WCNCPOpBatchWrite       WCNCPOperation = "BATCH_WRITE"
	WCNCPOpManifestWrite    WCNCPOperation = "MANIFEST_WRITE"
	WCNCPOpHeadCAS          WCNCPOperation = "HEAD_CAS"
	WCNCPOpSnapshotRead     WCNCPOperation = "SNAPSHOT_READ"
	WCNCPOpProposalClaim    WCNCPOperation = "PROPOSAL_CLAIM"
	WCNCPOpValidationWrite  WCNCPOperation = "VALIDATION_WRITE"
	WCNCPOpDecisionWrite    WCNCPOperation = "DECISION_WRITE"
)

var (
	// ErrWCNCPForbidden means the authenticated actor lacks product authority.
	ErrWCNCPForbidden = errors.New("BuildOpt WCNCP actor is not authorized")
	// ErrWCNCPForkReadOnly means a fork or pull-request runner attempted a
	// control-plane publication. Forks are read-only by default.
	ErrWCNCPForkReadOnly = errors.New("BuildOpt WCNCP fork runner is read-only")
)

// GrantWCNCPActor binds one existing central token to a product actor. The
// grant inherits token expiry and revocation; deleting the token cascades the
// grant. Actor capabilities are fixed by the frozen contract and cannot be
// widened per-grant.
func (storage *Storage) GrantWCNCPActor(ctx context.Context, tokenID string, actor WCNCPActor, now time.Time) (WCNCPGrant, error) {
	if storage == nil || ctx == nil || now.IsZero() {
		return WCNCPGrant{}, ErrWCNCPInvalid
	}
	grant, err := wcncpGrantForActor(tokenID, actor)
	if err != nil {
		return WCNCPGrant{}, err
	}
	finish, err := storage.beginOperation()
	if err != nil {
		return WCNCPGrant{}, err
	}
	defer finish()
	// The token must exist; the FK enforces it, but check first for a typed error.
	var exists int
	err = storage.control.database.QueryRowContext(ctx, `SELECT 1 FROM central_access_tokens WHERE token_id = ?`, tokenID).Scan(&exists)
	if err != nil {
		return WCNCPGrant{}, ErrWCNCPInvalid
	}
	observation, proposal, validation, decision := 0, 0, 0, 0
	if grant.ObservationWrite {
		observation = 1
	}
	if grant.ProposalClaim {
		proposal = 1
	}
	if grant.ValidationWrite {
		validation = 1
	}
	if grant.DecisionWrite {
		decision = 1
	}
	if _, err := storage.control.database.ExecContext(ctx, `INSERT INTO wcncp_actor_grants (
    token_id, actor, observation_write, proposal_claim, validation_write,
    decision_write, fork_write_allowed, issued_at_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, 0, ?)
ON CONFLICT(token_id) DO UPDATE SET
    actor = excluded.actor,
    observation_write = excluded.observation_write,
    proposal_claim = excluded.proposal_claim,
    validation_write = excluded.validation_write,
    decision_write = excluded.decision_write,
    issued_at_unix_ms = excluded.issued_at_unix_ms`,
		tokenID, string(actor), observation, proposal, validation, decision, now.UTC().UnixMilli()); err != nil {
		return WCNCPGrant{}, err
	}
	return grant, nil
}

func wcncpGrantForActor(tokenID string, actor WCNCPActor) (WCNCPGrant, error) {
	if len(tokenID) != 32 {
		return WCNCPGrant{}, ErrWCNCPInvalid
	}
	switch actor {
	case WCNCPActorDeveloper:
		return WCNCPGrant{TokenID: tokenID, Actor: actor}, nil
	case WCNCPActorTrustedObserver:
		return WCNCPGrant{TokenID: tokenID, Actor: actor, ObservationWrite: true}, nil
	case WCNCPActorValidator:
		return WCNCPGrant{TokenID: tokenID, Actor: actor, ProposalClaim: true, ValidationWrite: true}, nil
	case WCNCPActorOwner:
		return WCNCPGrant{TokenID: tokenID, Actor: actor, DecisionWrite: true}, nil
	case WCNCPActorAdmin:
		return WCNCPGrant{TokenID: tokenID, Actor: actor}, nil
	default:
		return WCNCPGrant{}, ErrWCNCPInvalid
	}
}

// authenticateWCNCP returns the transport authorization, the actor grant (if
// any), and the token identifier hash for audit. Raw credentials never leave
// this function and never enter logs or evidence.
func (storage *Storage) authenticateWCNCP(ctx context.Context, raw []byte, now time.Time) (CentralTokenAuthorization, WCNCPGrant, string, bool, error) {
	finish, err := storage.beginOperation()
	if err != nil {
		return CentralTokenAuthorization{}, WCNCPGrant{}, "", false, err
	}
	defer finish()
	authorization, authorized, err := authenticateCentralToken(ctx, storage.control.database, raw, now)
	if err != nil || !authorized {
		return CentralTokenAuthorization{}, WCNCPGrant{}, "", authorized, err
	}
	var (
		tokenID                           string
		actor                             sql.NullString
		observation, proposal, validation, decision sql.NullInt64
	)
	err = storage.control.database.QueryRowContext(ctx, `SELECT token.token_id, grant.actor, grant.observation_write,
       grant.proposal_claim, grant.validation_write, grant.decision_write
FROM central_access_tokens AS token
LEFT JOIN wcncp_actor_grants AS grant ON grant.token_id = token.token_id
WHERE token.token_digest = ?`, centralTokenDigest(raw)).Scan(
		&tokenID, &actor, &observation, &proposal, &validation, &decision)
	if err != nil {
		// No grant row means developer-default: transport only, no WCNCP writes.
		if errors.Is(err, sql.ErrNoRows) {
			return authorization, WCNCPGrant{Actor: WCNCPActorDeveloper}, "", true, nil
		}
		return CentralTokenAuthorization{}, WCNCPGrant{}, "", false, err
	}
	// LEFT JOIN with missing grant yields NULL actor; treat as developer.
	if !actor.Valid || actor.String == "" {
		return authorization, WCNCPGrant{TokenID: tokenID, Actor: WCNCPActorDeveloper}, tokenID, true, nil
	}
	return authorization, WCNCPGrant{
		TokenID: tokenID, Actor: WCNCPActor(actor.String),
		ObservationWrite: observation.Valid && observation.Int64 == 1, ProposalClaim: proposal.Valid && proposal.Int64 == 1,
		ValidationWrite: validation.Valid && validation.Int64 == 1, DecisionWrite: decision.Valid && decision.Int64 == 1,
	}, tokenID, true, nil
}

// AuthorizeWCNCPOperation enforces repository scope, actor capability, and
// fork read-only before any storage mutation. It never inspects credentials.
func AuthorizeWCNCPOperation(authorization CentralTokenAuthorization, grant WCNCPGrant, operation WCNCPOperation, repositoryScopeSHA256 string, kind StateKind, forked bool) error {
	if !validSHA256(repositoryScopeSHA256) || !validWCNCPKind(kind) {
		return ErrWCNCPInvalid
	}
	if authorization.Scope.RepositoryScopeSHA256 != repositoryScopeSHA256 {
		return ErrWCNCPForbidden
	}
	write := operation != WCNCPOpSnapshotRead
	if write {
		if !authorization.Has(CentralStateWrite) {
			return ErrWCNCPForbidden
		}
		if forked {
			return ErrWCNCPForkReadOnly
		}
	} else if !authorization.Has(CentralStateRead) && !authorization.Has(CentralStateWrite) {
		return ErrWCNCPForbidden
	}
	switch operation {
	case WCNCPOpObservationWrite, WCNCPOpBatchWrite:
		if kind != WCNCPKindObservation || !grant.ObservationWrite {
			return ErrWCNCPForbidden
		}
	case WCNCPOpManifestWrite, WCNCPOpHeadCAS:
		switch kind {
		case WCNCPKindObservation:
			if !grant.ObservationWrite {
				return ErrWCNCPForbidden
			}
		case WCNCPKindOpportunity:
			// Opportunities are derived server-side in WCNCP-004; direct
			// publication is closed in WCNCP-002.
			return ErrWCNCPForbidden
		case WCNCPKindProposal:
			if !grant.ProposalClaim {
				return ErrWCNCPForbidden
			}
		case WCNCPKindValidation:
			if !grant.ValidationWrite {
				return ErrWCNCPForbidden
			}
		case WCNCPKindDecision:
			if !grant.DecisionWrite {
				return ErrWCNCPForbidden
			}
		default:
			return ErrWCNCPInvalid
		}
	case WCNCPOpSnapshotRead:
		return nil
	case WCNCPOpProposalClaim:
		if kind != WCNCPKindProposal || !grant.ProposalClaim {
			return ErrWCNCPForbidden
		}
	case WCNCPOpValidationWrite:
		if kind != WCNCPKindValidation || !grant.ValidationWrite {
			return ErrWCNCPForbidden
		}
	case WCNCPOpDecisionWrite:
		if kind != WCNCPKindDecision || !grant.DecisionWrite {
			return ErrWCNCPForbidden
		}
		if grant.Actor != WCNCPActorOwner {
			// Even a validator with validation rights cannot accept.
			return ErrWCNCPForbidden
		}
	default:
		return ErrWCNCPInvalid
	}
	return nil
}

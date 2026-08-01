package buildsession

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tonyredondo/buildopt/internal/datalifecycle"
)

const (
	exportEventSchemaVersion  = "1.0"
	exportJSONLMaximumBytes   = 64 << 20
	exportJSONLMaximumLine    = 1 << 20
	observedEventSequence     = 1
	publishedEventSequence    = 2
	observedEventPayloadType  = "BUILD_SESSION_OBSERVED"
	publishedEventPayloadType = "BUILD_SESSION_PUBLISHED"
)

var (
	errJSONLEventConflict = errors.New(
		"JSONL event identity or sequence was reused with different content",
	)
	errJSONLFull                 = errors.New("JSONL export stream reached its byte limit")
	exportEventIdentifierPattern = regexp.MustCompile(
		`^[A-Za-z0-9][A-Za-z0-9._:/@+-]{0,255}$`,
	)
)

type exportEvent struct {
	EventID        string          `json:"eventId"`
	BuildID        string          `json:"buildId"`
	Sequence       int             `json:"sequence"`
	OccurredAt     string          `json:"occurredAt"`
	EmittedAt      string          `json:"emittedAt"`
	SchemaVersion  string          `json:"schemaVersion"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Profile        string          `json:"profile"`
	Payload        json.RawMessage `json:"payload"`
}

type observedEventPayload struct {
	Type     string   `json:"type"`
	Document Document `json:"document"`
}

type publishedEventPayload struct {
	Type           string `json:"type"`
	Filename       string `json:"filename"`
	DocumentDigest string `json:"documentDigest"`
}

type jsonlStream struct {
	path              string
	maximumBytes      int64
	authorizedProfile datalifecycle.ExportProfile
}

type storedExportEvent struct {
	event exportEvent
	line  []byte
}

func newBuildSessionEvents(
	document Document,
	target string,
	documentBytes []byte,
	profile datalifecycle.ExportProfile,
) ([2]exportEvent, error) {
	observedPayload, err := json.Marshal(observedEventPayload{
		Type:     observedEventPayloadType,
		Document: document,
	})
	if err != nil {
		return [2]exportEvent{}, errors.New(
			"encode observed BUILD_SESSION JSONL payload",
		)
	}
	documentDigest := sha256.Sum256(documentBytes)
	publishedPayload, err := json.Marshal(publishedEventPayload{
		Type:           publishedEventPayloadType,
		Filename:       filepath.Base(target),
		DocumentDigest: "sha256:" + hex.EncodeToString(documentDigest[:]),
	})
	if err != nil {
		return [2]exportEvent{}, errors.New(
			"encode published BUILD_SESSION JSONL payload",
		)
	}
	occurredAt := document.Build.CompletedAt
	return [2]exportEvent{
		newExportEvent(
			document.Build.ID,
			observedEventSequence,
			occurredAt,
			observedPayload,
			profile,
		),
		newExportEvent(
			document.Build.ID,
			publishedEventSequence,
			occurredAt,
			publishedPayload,
			profile,
		),
	}, nil
}

func newExportEvent(
	buildID string,
	sequence int,
	occurredAt string,
	payload []byte,
	profile datalifecycle.ExportProfile,
) exportEvent {
	sum := sha256.New()
	_, _ = sum.Write([]byte("buildopt-export-event-v1"))
	_, _ = sum.Write([]byte{0})
	_, _ = sum.Write([]byte(buildID))
	_, _ = sum.Write([]byte{0})
	_, _ = sum.Write([]byte(strconv.Itoa(sequence)))
	_, _ = sum.Write([]byte{0})
	_, _ = sum.Write(payload)
	return exportEvent{
		EventID:       "event-" + hex.EncodeToString(sum.Sum(nil)),
		BuildID:       buildID,
		Sequence:      sequence,
		OccurredAt:    occurredAt,
		EmittedAt:     occurredAt,
		SchemaVersion: exportEventSchemaVersion,
		IdempotencyKey: buildID + "/" +
			strconv.Itoa(sequence),
		Profile: string(profile),
		Payload: append(json.RawMessage(nil), payload...),
	}
}

func (stream *jsonlStream) append(event exportEvent) error {
	line, err := encodeExportEventLine(event)
	if err != nil {
		return err
	}

	maximum := stream.maximum()
	file, err := openNoFollow(
		stream.path,
		os.O_CREATE|os.O_EXCL|os.O_APPEND|os.O_WRONLY,
		0o600,
	)
	created := err == nil
	if errors.Is(err, fs.ErrExist) {
		file, err = openNoFollow(
			stream.path,
			os.O_APPEND|os.O_WRONLY,
			0,
		)
	}
	if err != nil {
		return errors.New("open JSONL export stream")
	}
	defer file.Close()
	info, err := trustedJSONLFile(file, maximum)
	if err != nil {
		return err
	}
	if info.Size()+int64(len(line)) > maximum {
		return errJSONLFull
	}
	written, err := file.Write(line)
	if err != nil || written != len(line) {
		return errors.New("append complete JSONL export event")
	}
	if err := file.Sync(); err != nil {
		return errors.New("sync JSONL export stream")
	}
	if created {
		if err := syncDirectory(filepath.Dir(stream.path)); err != nil {
			return err
		}
	}
	return nil
}

func (stream *jsonlStream) ensureCapacity(events ...exportEvent) error {
	additionalBytes := int64(0)
	for _, event := range events {
		line, err := encodeExportEventLine(event)
		if err != nil {
			return err
		}
		additionalBytes += int64(len(line))
	}
	raw, _, err := stream.load(true)
	if err != nil {
		return err
	}
	if int64(len(raw))+additionalBytes > stream.maximum() {
		return errJSONLFull
	}
	return nil
}

func encodeExportEventLine(event exportEvent) ([]byte, error) {
	if err := validateExportEvent(event); err != nil {
		return nil, err
	}
	line, err := json.Marshal(event)
	if err != nil {
		return nil, errors.New("encode JSONL export event")
	}
	if len(line)+1 > exportJSONLMaximumLine {
		return nil, errors.New("JSONL export event exceeds its line limit")
	}
	line = append(line, '\n')
	return line, nil
}

func (stream *jsonlStream) load(
	repairTrailingLine bool,
) ([]byte, map[string]map[int]storedExportEvent, error) {
	maximum := stream.maximum()
	file, err := openNoFollow(
		stream.path,
		os.O_RDWR,
		0,
	)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, map[string]map[int]storedExportEvent{}, nil
	}
	if err != nil {
		return nil, nil, errors.New("open JSONL export stream")
	}
	defer file.Close()
	info, err := trustedJSONLFile(file, maximum)
	if err != nil {
		return nil, nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil ||
		int64(len(raw)) != info.Size() ||
		int64(len(raw)) > maximum {
		return nil, nil, errors.New("read bounded JSONL export stream")
	}
	if len(raw) != 0 && raw[len(raw)-1] != '\n' {
		if !repairTrailingLine {
			return nil, nil, errors.New(
				"JSONL export stream has a truncated trailing event",
			)
		}
		validLength := bytes.LastIndexByte(raw, '\n') + 1
		if err := file.Truncate(int64(validLength)); err != nil {
			return nil, nil, errors.New(
				"truncate incomplete JSONL export event",
			)
		}
		if err := file.Sync(); err != nil {
			return nil, nil, errors.New(
				"sync repaired JSONL export stream",
			)
		}
		raw = raw[:validLength]
	}
	events, err := decodeJSONLEvents(raw, stream.authorizedProfile)
	if err != nil {
		return nil, nil, err
	}
	return raw, events, nil
}

func (stream *jsonlStream) maximum() int64 {
	if stream.maximumBytes > 0 {
		return stream.maximumBytes
	}
	return exportJSONLMaximumBytes
}

func trustedJSONLFile(
	file *os.File,
	maximum int64,
) (os.FileInfo, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, errors.New("inspect JSONL export stream")
	}
	if !privateFileInfo(info) ||
		info.Size() < 0 ||
		info.Size() > maximum {
		return nil, errors.New(
			"JSONL export stream is not a bounded private regular file",
		)
	}
	return info, nil
}

func decodeJSONLEvents(
	raw []byte,
	authorizedProfile datalifecycle.ExportProfile,
) (map[string]map[int]storedExportEvent, error) {
	byEventID := make(map[string][]byte)
	byBuild := make(map[string]map[int]storedExportEvent)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), exportJSONLMaximumLine)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		event, err := decodeExportEvent(line)
		if err != nil {
			return nil, err
		}
		if exportProfileRank(datalifecycle.ExportProfile(event.Profile)) >
			exportProfileRank(authorizedProfile) {
			return nil, errors.New(
				"JSONL export profile exceeds current authorization",
			)
		}
		if previous, exists := byEventID[event.EventID]; exists {
			if !bytes.Equal(previous, line) {
				return nil, errJSONLEventConflict
			}
			continue
		}
		byEventID[event.EventID] = line
		sequences := byBuild[event.BuildID]
		if sequences == nil {
			sequences = make(map[int]storedExportEvent)
			byBuild[event.BuildID] = sequences
		}
		if previous, exists := sequences[event.Sequence]; exists {
			if !bytes.Equal(previous.line, line) {
				return nil, errJSONLEventConflict
			}
			continue
		}
		sequences[event.Sequence] = storedExportEvent{
			event: event,
			line:  line,
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.New("scan JSONL export stream")
	}
	return byBuild, nil
}

func decodeExportEvent(line []byte) (exportEvent, error) {
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	var event exportEvent
	if err := decoder.Decode(&event); err != nil {
		return exportEvent{}, errors.New("decode JSONL export event")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return exportEvent{}, errors.New(
			"JSONL export line must contain one event",
		)
	}
	if err := validateExportEvent(event); err != nil {
		return exportEvent{}, err
	}
	expected := newExportEvent(
		event.BuildID,
		event.Sequence,
		event.OccurredAt,
		event.Payload,
		datalifecycle.ExportProfile(event.Profile),
	)
	if event.EventID != expected.EventID ||
		event.IdempotencyKey != expected.IdempotencyKey ||
		event.EmittedAt != expected.EmittedAt {
		return exportEvent{}, errors.New(
			"JSONL export event identity is not deterministic",
		)
	}
	return event, nil
}

func validateExportEvent(event exportEvent) error {
	if !exportEventIdentifierPattern.MatchString(event.EventID) ||
		!exportEventIdentifierPattern.MatchString(event.BuildID) ||
		event.SchemaVersion != exportEventSchemaVersion ||
		exportProfileRank(datalifecycle.ExportProfile(event.Profile)) < 0 ||
		event.Sequence < observedEventSequence ||
		event.Sequence > publishedEventSequence ||
		event.IdempotencyKey != event.BuildID+"/"+
			strconv.Itoa(event.Sequence) ||
		len(event.Payload) == 0 ||
		len(event.Payload) > exportJSONLMaximumLine {
		return errors.New("invalid JSONL export event")
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, event.OccurredAt)
	if err != nil || occurredAt.Location() != time.UTC {
		return errors.New("JSONL event occurredAt must be UTC RFC3339")
	}
	emittedAt, err := time.Parse(time.RFC3339Nano, event.EmittedAt)
	if err != nil || emittedAt.Location() != time.UTC ||
		emittedAt.Before(occurredAt) {
		return errors.New("JSONL event emittedAt must be ordered UTC RFC3339")
	}

	switch event.Sequence {
	case observedEventSequence:
		var payload observedEventPayload
		if err := decodeStrictPayload(event.Payload, &payload); err != nil ||
			payload.Type != observedEventPayloadType ||
			!payload.Document.Complete ||
			payload.Document.Recovery != nil ||
			payload.Document.RecordType != recordType ||
			payload.Document.SchemaVersion != schemaVersion ||
			payload.Document.Build.ID != event.BuildID {
			return errors.New("invalid observed BUILD_SESSION JSONL payload")
		}
	case publishedEventSequence:
		var payload publishedEventPayload
		if err := decodeStrictPayload(event.Payload, &payload); err != nil ||
			payload.Type != publishedEventPayloadType ||
			payload.Filename != exportFilename(event.BuildID) ||
			!isSHA256Digest(payload.DocumentDigest) {
			return errors.New("invalid published BUILD_SESSION JSONL payload")
		}
	}
	return nil
}

func exportProfileRank(profile datalifecycle.ExportProfile) int {
	switch profile {
	case datalifecycle.ExportSummary:
		return 0
	case datalifecycle.ExportTasks:
		return 1
	case datalifecycle.ExportEvidence:
		return 2
	case datalifecycle.ExportDiagnostic:
		return 3
	default:
		return -1
	}
}

func isSHA256Digest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") ||
		len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(value[len("sha256:"):])
	return err == nil
}

func decodeStrictPayload(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("JSONL payload contains trailing data")
	}
	return nil
}

func (exporter *Exporter) recoverPartialDocuments() error {
	exporter.mutex.Lock()
	defer exporter.mutex.Unlock()
	_, builds, err := exporter.stream.load(true)
	if err != nil {
		return err
	}
	for buildID, events := range builds {
		observed, hasObserved := events[observedEventSequence]
		published, hasPublished := events[publishedEventSequence]
		if !hasObserved {
			return errors.New(
				"JSONL build sequence has no observed BUILD_SESSION",
			)
		}
		var observedPayload observedEventPayload
		if err := decodeStrictPayload(
			observed.event.Payload,
			&observedPayload,
		); err != nil {
			return err
		}
		partialPath := filepath.Join(
			exporter.directory,
			partialExportFilename(buildID, observed.event.EventID),
		)
		partialExists, partialContent, err := readOptionalPrivateFile(
			partialPath,
		)
		if err != nil {
			return err
		}
		if partialExists {
			if err := validateRecoveredDocument(
				observedPayload.Document,
				partialContent,
			); err != nil {
				return err
			}
		}
		completePath := filepath.Join(
			exporter.directory,
			exportFilename(buildID),
		)
		completeExists, completeContent, err := readOptionalPrivateFile(
			completePath,
		)
		if err != nil {
			return err
		}
		if hasPublished {
			if !completeExists {
				return errors.New(
					"JSONL publication event has no immutable BUILD_SESSION",
				)
			}
			if err := validatePublishedDocument(
				published.event,
				completePath,
				completeContent,
				observedPayload.Document,
			); err != nil {
				return err
			}
			continue
		}
		if completeExists {
			if err := validateObservedDocument(
				observedPayload.Document,
				completeContent,
			); err != nil {
				return err
			}
			events, err := newBuildSessionEvents(
				observedPayload.Document,
				completePath,
				completeContent,
				datalifecycle.ExportProfile(observed.event.Profile),
			)
			if err != nil {
				return err
			}
			if err := exporter.stream.append(
				events[publishedEventSequence-1],
			); err != nil {
				return err
			}
			continue
		}

		if partialExists {
			continue
		}
		recovered, err := RecoverPartial(
			observedPayload.Document,
			exporter.now().UTC(),
			[]MissingSequenceRange{{
				First: publishedEventSequence,
				Last:  publishedEventSequence,
			}},
		)
		if err != nil {
			return err
		}
		content, err := json.MarshalIndent(recovered, "", "  ")
		if err != nil {
			return errors.New("encode recovered partial BUILD_SESSION")
		}
		content = append(content, '\n')
		if err := publishPrivateFile(
			exporter.directory,
			partialPath,
			content,
			"partial BUILD_SESSION recovery",
		); err != nil {
			return err
		}
	}
	return nil
}

func validateObservedDocument(
	observed Document,
	content []byte,
) error {
	expected, err := json.MarshalIndent(observed, "", "  ")
	if err != nil {
		return errors.New("encode observed BUILD_SESSION")
	}
	expected = append(expected, '\n')
	if !bytes.Equal(content, expected) {
		return errors.New(
			"complete BUILD_SESSION does not match its observed JSONL event",
		)
	}
	return nil
}

func validateRecoveredDocument(
	observed Document,
	content []byte,
) error {
	var document Document
	if err := decodeStrictPayload(content, &document); err != nil ||
		document.Complete ||
		document.Recovery == nil {
		return errors.New("recovered partial BUILD_SESSION is invalid")
	}
	recoveredAt, err := time.Parse(
		time.RFC3339Nano,
		document.Recovery.RecoveredAt,
	)
	if err != nil {
		return errors.New("recovered partial BUILD_SESSION is invalid")
	}
	expected, err := RecoverPartial(
		observed,
		recoveredAt,
		[]MissingSequenceRange{{
			First: publishedEventSequence,
			Last:  publishedEventSequence,
		}},
	)
	if err != nil {
		return errors.New("recovered partial BUILD_SESSION is invalid")
	}
	expectedContent, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		return errors.New("encode recovered partial BUILD_SESSION")
	}
	expectedContent = append(expectedContent, '\n')
	if !bytes.Equal(content, expectedContent) {
		return errors.New(
			"recovered partial BUILD_SESSION does not match its observed event",
		)
	}
	return nil
}

func validatePublishedDocument(
	event exportEvent,
	path string,
	content []byte,
	observed Document,
) error {
	var payload publishedEventPayload
	if err := decodeStrictPayload(event.Payload, &payload); err != nil {
		return err
	}
	sum := sha256.Sum256(content)
	if payload.Filename != filepath.Base(path) ||
		payload.DocumentDigest !=
			"sha256:"+hex.EncodeToString(sum[:]) {
		return errors.New(
			"published JSONL event does not bind the immutable BUILD_SESSION",
		)
	}
	var document Document
	if err := decodeStrictPayload(content, &document); err != nil ||
		!document.Complete ||
		document.Recovery != nil ||
		document.Build.ID != event.BuildID {
		return errors.New("published BUILD_SESSION is invalid")
	}
	if err := validateObservedDocument(observed, content); err != nil {
		return err
	}
	return nil
}

func readOptionalPrivateFile(path string) (bool, []byte, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil, nil
	}
	if err != nil ||
		!privateFileInfo(info) ||
		info.Size() < 0 ||
		info.Size() > exportJSONLMaximumLine {
		return false, nil, errors.New(
			"export document is not a bounded private regular file",
		)
	}
	content, err := os.ReadFile(path)
	if err != nil || int64(len(content)) != info.Size() {
		return false, nil, errors.New("read complete export document")
	}
	return true, content, nil
}

func partialExportFilename(buildID string, eventID string) string {
	sum := sha256.Sum256([]byte(buildID + "\x00" + eventID))
	return "build-session-" + hex.EncodeToString(sum[:]) + "-partial.json"
}

package buildhistory

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode"
)

// Handler serves the bounded loopback history API.
type Handler struct {
	token string
	store *Store
}

// NewHandler creates a history handler with an independent read credential.
func NewHandler(token string, directory string) (*Handler, error) {
	if err := validateToken(token); err != nil {
		return nil, err
	}
	store, err := NewStore(directory)
	if err != nil {
		return nil, err
	}
	return &Handler{token: token, store: store}, nil
}

// ServeHTTP authenticates before routing so absent identities and errors do
// not disclose history metadata.
func (handler *Handler) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	if !handler.authenticate(request) {
		writer.Header().Set(
			"WWW-Authenticate",
			`Bearer realm="buildopt-server-history"`,
		)
		http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	switch request.URL.Path {
	case ListPath:
		handler.list(writer, request)
	case DetailPath:
		handler.detail(writer, request)
	default:
		http.NotFound(writer, request)
	}
}

func (handler *Handler) list(
	writer http.ResponseWriter,
	request *http.Request,
) {
	values := request.URL.Query()
	if invalidQueryParameter(values, "repository", "outcome", "limit", "cursor") {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	limit, err := parseLimit(queryValue(values, "limit"))
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	response, err := handler.store.List(Filter{
		RepositoryID: queryValue(values, "repository"),
		Outcome:      queryValue(values, "outcome"),
		Limit:        limit,
		Cursor:       queryValue(values, "cursor"),
	})
	if errors.Is(err, ErrInvalidQuery) {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writeJSON(writer, response)
}

func (handler *Handler) detail(
	writer http.ResponseWriter,
	request *http.Request,
) {
	values := request.URL.Query()
	if invalidQueryParameter(values, "id") || queryValue(values, "id") == "" {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	response, err := handler.store.Get(queryValue(values, "id"))
	if errors.Is(err, ErrInvalidQuery) {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if errors.Is(err, ErrNotFound) {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writeJSON(writer, response)
}

func (handler *Handler) authenticate(request *http.Request) bool {
	values := request.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return false
	}
	actual := strings.TrimPrefix(values[0], "Bearer ")
	return subtle.ConstantTimeCompare([]byte(actual), []byte(handler.token)) == 1
}

func writeJSON(writer http.ResponseWriter, value any) {
	content, err := json.Marshal(value)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(append(content, '\n'))
}

func validateToken(token string) error {
	if len(token) < 32 || len(token) > 512 {
		return errors.New("build history token must contain 32 to 512 bytes")
	}
	for _, character := range token {
		if unicode.IsControl(character) || unicode.IsSpace(character) {
			return errors.New("build history token contains whitespace or control characters")
		}
	}
	return nil
}
